package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type RegisterBank struct {
	mu        sync.RWMutex
	registers [100]uint16
}

func envString(name string, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}

func envFloat(name string, fallback float64) float64 {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func envInt(name string, fallback int) int {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func scaled(value float64, scale float64) uint16 {
	if value < 0 {
		return 0
	}
	return uint16(math.Round(value / scale))
}

func (bank *RegisterBank) set(values []uint16) {
	bank.mu.Lock()
	defer bank.mu.Unlock()
	copy(bank.registers[:], values)
}

func (bank *RegisterBank) read(address uint16, count uint16) ([]uint16, bool) {
	bank.mu.RLock()
	defer bank.mu.RUnlock()
	end := int(address) + int(count)
	if count == 0 || end > len(bank.registers) {
		return nil, false
	}
	values := make([]uint16, count)
	copy(values, bank.registers[address:uint16(end)])
	return values, true
}

func generatorValue(mode string, elapsed float64, base float64, index int) float64 {
	switch mode {
	case "random-int":
		return float64(int(base + rand.Float64()*100))
	case "sawtooth":
		return base + math.Mod(elapsed*float64(index+1), 60)
	case "square":
		if int(elapsed/5)%2 == 0 {
			return base + 10
		}
		return base - 10
	case "triangle":
		period := 30.0
		phase := math.Mod(elapsed, period) / period
		if phase < 0.5 {
			return base + phase*20
		}
		return base + (1-phase)*20
	case "random-string":
		return float64(len([]string{"idle", "running", "blocked", "maintenance"}[rand.Intn(4)]))
	case "array":
		return base + float64(index) + rand.Float64()*4
	default:
		return base + math.Sin(elapsed/(9.0+float64(index)*3))*10.0 + rand.Float64()*0.4 - 0.2
	}
}

func updateRegisters(bank *RegisterBank, tempBase float64, humidityBase float64, pressureBase float64, vibrationBase float64, generatorMode string) {
	start := time.Now()
	cycle := uint16(0)
	for {
		elapsed := time.Since(start).Seconds()
		temp := generatorValue(generatorMode, elapsed, tempBase, 0)
		humidity := generatorValue(generatorMode, elapsed, humidityBase, 1)
		pressure := generatorValue(generatorMode, elapsed, pressureBase, 2) / 25.0
		vibration := math.Abs(generatorValue(generatorMode, elapsed, vibrationBase, 3)) / 70.0

		status := uint16(0)
		if temp > 29.0 || vibration > 0.95 {
			status = 1
		}
		if temp > 32.0 || vibration > 1.15 {
			status = 2
		}

		bank.set([]uint16{scaled(temp, 0.1), scaled(humidity, 0.1), scaled(pressure, 0.01), scaled(vibration, 0.01), status, cycle})
		cycle++
		time.Sleep(time.Second)
	}
}

func exceptionResponse(transactionID uint16, unitID byte, functionCode byte, exception byte) []byte {
	pdu := []byte{functionCode | 0x80, exception}
	header := make([]byte, 7)
	binary.BigEndian.PutUint16(header[0:2], transactionID)
	binary.BigEndian.PutUint16(header[2:4], 0)
	binary.BigEndian.PutUint16(header[4:6], uint16(len(pdu)+1))
	header[6] = unitID
	return append(header, pdu...)
}

func handleConnection(conn net.Conn, bank *RegisterBank) {
	defer conn.Close()
	for {
		header := make([]byte, 7)
		if _, err := io.ReadFull(conn, header); err != nil {
			return
		}
		transactionID := binary.BigEndian.Uint16(header[0:2])
		length := binary.BigEndian.Uint16(header[4:6])
		unitID := header[6]
		if length < 2 {
			return
		}
		pdu := make([]byte, int(length)-1)
		if _, err := io.ReadFull(conn, pdu); err != nil {
			return
		}
		if len(pdu) < 5 {
			_, _ = conn.Write(exceptionResponse(transactionID, unitID, pdu[0], 3))
			continue
		}

		functionCode := pdu[0]
		if functionCode != 3 {
			_, _ = conn.Write(exceptionResponse(transactionID, unitID, functionCode, 1))
			continue
		}
		address := binary.BigEndian.Uint16(pdu[1:3])
		count := binary.BigEndian.Uint16(pdu[3:5])
		values, ok := bank.read(address, count)
		if !ok || count > 60 {
			_, _ = conn.Write(exceptionResponse(transactionID, unitID, functionCode, 2))
			continue
		}

		responsePDU := make([]byte, 2+len(values)*2)
		responsePDU[0] = functionCode
		responsePDU[1] = byte(len(values) * 2)
		for index, value := range values {
			binary.BigEndian.PutUint16(responsePDU[2+index*2:4+index*2], value)
		}
		responseHeader := make([]byte, 7)
		binary.BigEndian.PutUint16(responseHeader[0:2], transactionID)
		binary.BigEndian.PutUint16(responseHeader[2:4], 0)
		binary.BigEndian.PutUint16(responseHeader[4:6], uint16(len(responsePDU)+1))
		responseHeader[6] = unitID
		_, _ = conn.Write(append(responseHeader, responsePDU...))
	}
}

func startUI(deviceID string, protocol string, port int, uiPort int, generatorMode string, startedAt time.Time, bank *RegisterBank) {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html><html><head><meta charset="utf-8"><title>IoT Presense</title><style>
:root{--bg:#14161a;--side:#101216;--panel:#1b1f26;--border:#2a2f38;--text:#e7e9ee;--muted:#8a9099;--accent:#8a9099;--green:#28c76f;--yellow:#febe3b;--red:#ff4d4f}
*{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--text);font-family:Inter,system-ui,Segoe UI,Arial,sans-serif}.layout{display:grid;grid-template-columns:250px 1fr;min-height:100vh}aside{background:var(--side);border-right:1px solid var(--border);padding:18px}.brand{font-size:22px;font-weight:800;margin-bottom:22px}.brand span{color:var(--accent)}a{display:block;color:var(--text);text-decoration:none;padding:11px 12px;border-radius:10px;margin:6px 0;border:1px solid transparent}a.active,a:hover{background:var(--panel);border-color:var(--border)}main{padding:22px}.card{background:var(--panel);border:1px solid var(--border);border-radius:16px;padding:16px;margin-bottom:14px}.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(260px,1fr));gap:14px}.k{color:var(--muted);font-size:12px}.v{font-size:24px;font-weight:800;margin-top:6px}pre{background:#0e0f12;border:1px solid var(--border);border-radius:12px;padding:12px;overflow:auto}.hidden{display:none}.bar{height:10px;background:#0e0f12;border:1px solid var(--border);border-radius:999px;overflow:hidden}.bar span{display:block;height:100%;background:var(--accent)}.toolbar{display:flex;gap:10px;flex-wrap:wrap;margin-bottom:14px}label{display:inline-flex;align-items:center;gap:6px;background:#0e0f12;border:1px solid var(--border);border-radius:999px;padding:8px 11px}.log-row{background:#0e0f12;border:1px solid var(--border);border-radius:14px;padding:12px;margin:10px 0;font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace}.log-info{color:var(--green)}.log-warn{color:var(--yellow)}.log-error{color:var(--red)}.dot{display:inline-block;width:10px;height:10px;border-radius:50%;margin-right:8px;background:currentColor}.overview-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:12px;margin-bottom:14px}.overview-card{background:#11151b;border:1px solid var(--border);border-radius:14px;padding:12px}.overview-value{font-size:20px;font-weight:800;margin-top:6px}.overview-hint{color:var(--muted);font-size:12px;margin-top:5px}.health-layout{display:grid;grid-template-columns:360px 1fr;gap:14px;margin-bottom:14px}.health-pie{width:190px;height:190px;border-radius:50%;background:var(--green);margin:22px auto 10px}.health-item{display:flex;justify-content:space-between;border-bottom:1px solid var(--border);padding:8px 0}.connection-graph{position:relative;min-height:260px;height:260px;overflow:hidden;background:#11151b;border:1px solid var(--border);border-radius:16px;margin-top:12px}.edge-svg{position:absolute;left:0;top:0;width:100%;height:100%;pointer-events:none}.node{position:absolute;min-width:150px;border:1px solid var(--accent);border-radius:14px;background:var(--panel);padding:10px 12px;box-shadow:0 8px 20px rgba(0,0,0,.25);z-index:2}.node .title{font-weight:800;font-size:15px}.node .sub{color:var(--muted);font-size:12px;margin-top:4px}
</style></head><body><div class="layout"><aside><div class="brand">IoT <span>Presense</span></div><a href="#dashboard">Dashboard</a><a href="#metrics">Quick Metrics</a><a href="#components">Components</a><a href="#config">Configuration</a><a href="#logs">Logs</a></aside><main>
<section id="page-dashboard"><div id="cards"></div><div class="health-layout"><div class="card"><h3>Health distribution</h3><div class="health-pie"></div><div class="k" style="text-align:center">healthy: 1<br>unhealthy: 0</div></div><div class="card"><h3>Health details</h3><div id="healthDetails"></div></div></div><div class="card"><h3>Connection graph</h3><div id="connectionGraph" class="connection-graph"></div></div><div class="card"><h3>Current values</h3><pre id="values"></pre></div></section>
<section id="page-metrics" class="hidden"><div class="card"><h3>Quick Metrics</h3><div id="metricBars"></div></div></section>
<section id="page-components" class="hidden"><div class="card"><h3>Components</h3><div id="components"></div></div></section>
<section id="page-config" class="hidden"><div class="card"><h3>Configuration</h3><div class="toolbar"><button onclick="loadConfigPage()">Refresh configuration</button><button onclick="validateConfigDraft()">Validate changes</button><button id="applyConfigButton" disabled>Apply proposal</button><button onclick="resetConfigDraft()">Reset editor</button><span id="configState" class="k">Idle</span></div><textarea id="configEditor" spellcheck="false" style="width:100%;min-height:360px;background:#0e0f12;color:var(--text);border:1px solid var(--border);border-radius:12px;padding:12px;font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace"></textarea><h3>Validation and diff</h3><pre id="configDiff"></pre><h3>Last 5 configs</h3><div class="k">Backend write-back is not enabled yet.</div></div></section>
<section id="page-logs" class="hidden"><div class="card"><h3>Logs</h3><div class="toolbar"><label><input type="checkbox" class="log-filter" value="INFO" checked> Info</label><label><input type="checkbox" class="log-filter" value="WARN" checked> Warning</label><label><input type="checkbox" class="log-filter" value="ERROR" checked> Error</label></div><div id="logs"></div></div></section>
</main></div><script>
let status={},vals={};
const logsData=[
  {level:'INFO',message:'Presense simulator running'},
  {level:'INFO',message:'Generator mode active'},
  {level:'INFO',message:'Protocol endpoint ready'}
];
function esc(v){return String(v).replaceAll('&','&amp;').replaceAll('<','&lt;').replaceAll('>','&gt;').replaceAll('"','&quot;').replaceAll("'","&#039;")}
async function j(u){const r=await fetch(u);return r.json()}
function route(){const p=location.hash.replace("#","")||"dashboard";["dashboard","metrics","components","config","logs"].forEach(x=>document.getElementById("page-"+x).classList.toggle("hidden",x!==p));document.querySelectorAll("aside a").forEach(a=>a.classList.toggle("active",a.getAttribute("href")==="#"+p));render();if(p==="config")loadConfigPage()}
let configOriginal='';
function defaultRegisterConfig(){
  const metrics=(vals.metrics||{temperature:null,humidity:null,pressure:null,vibration:null,status:null,cycle:null});
  return Object.keys(metrics).map((name,index)=>({name:name,register:index,scale:name==='pressure'||name==='vibration'?0.01:name==='status'||name==='cycle'?1:0.1,unit:name==='temperature'?'°C':name==='humidity'?'%':name==='pressure'?'bar':name==='vibration'?'mm/s':'',generatorMode:status.generatorMode||'sine'}));
}
function currentConfig(){
  return {
    deviceId:status.deviceId||'unknown',
    generatorMode:status.generatorMode||'sine',
    host:status.host||status.deviceId||'localhost',
    port:Number(status.port||5020),
    protocol:status.protocol||'modbus-tcp',
    uiPort:Number(status.uiPort||status.port||0),
    supportedGenerators:status.supportedGenerators||['sine','random-int','random-string','array','sawtooth','square','triangle'],
    registers:defaultRegisterConfig(),
    currentValues:vals
  };
}
async function loadConfigPage(){
  try{
    if(!status.deviceId)status=await j('/api/status');
    if(!Object.keys(vals||{}).length)vals=await j('/api/values');
  }catch(e){
    logsData.push({level:'ERROR',message:e.message});
  }
  configOriginal=JSON.stringify(currentConfig(),null,2);
  configEditor.value=configOriginal;
  configDiff.textContent='';
  configState.textContent='Loaded current configuration';
  applyConfigButton.disabled=true;
}function diffText(a,b){const x=a.split('\n'),y=b.split('\n'),m=Math.max(x.length,y.length),o=[];for(let i=0;i<m;i++){if(x[i]===y[i]&&x[i]!==undefined)o.push('  '+x[i]);else{if(x[i]!==undefined)o.push('- '+x[i]);if(y[i]!==undefined)o.push('+ '+y[i]);}}return o.join('\n')}function validateConfigDraft(){try{const normalized=JSON.stringify(JSON.parse(configEditor.value),null,2);configDiff.textContent=diffText(configOriginal,normalized);configState.textContent='Draft is valid JSON';applyConfigButton.disabled=false;}catch(e){configDiff.textContent=e.message;configState.textContent='Validation failed';applyConfigButton.disabled=true;}}function resetConfigDraft(){configEditor.value=configOriginal;configDiff.textContent='';configState.textContent='Editor reset';applyConfigButton.disabled=true;}function pct(v){return Math.max(0,Math.min(100,Number(v)||0))}
function logClass(level){return level==='ERROR'?'log-error':level==='WARN'?'log-warn':'log-info'}
function renderLogs(){const selected=new Set(Array.from(document.querySelectorAll('.log-filter:checked')).map(x=>x.value));logs.innerHTML=logsData.filter(x=>selected.has(x.level)).map(x=>'<div class="log-row"><span class="'+logClass(x.level)+'"><span class="dot"></span>'+esc(x.level)+'</span> '+esc(x.message)+'</div>').join('')}

function renderPresenseGraph(){
  const box=document.getElementById('connectionGraph'); if(!box)return;
  const w=Math.max(760, box.clientWidth||760), h=260; box.style.height=h+'px';
  const leftX=40, midX=Math.round(w/2)-110, rightX=w-230, y=95;
  box.innerHTML='<svg class="edge-svg" width="'+w+'" height="'+h+'" viewBox="0 0 '+w+' '+h+'"></svg>';
  const svg=box.querySelector('svg');
  function line(x1,y1,x2,y2){const p=document.createElementNS('http://www.w3.org/2000/svg','path');const dx=Math.max(80,Math.abs(x2-x1)*0.45);p.setAttribute('d','M '+x1+' '+y1+' C '+(x1+dx)+' '+y1+', '+(x2-dx)+' '+y2+', '+x2+' '+y2);p.setAttribute('stroke','#8a9099');p.setAttribute('stroke-width','3');p.setAttribute('fill','none');svg.appendChild(p);}
  function node(x,y,title,sub){const n=document.createElement('div');n.className='node';n.style.left=x+'px';n.style.top=y+'px';n.style.width='200px';n.innerHTML='<div class="title">'+esc(title)+'</div><div class="sub">'+esc(sub||'')+'</div>';box.appendChild(n);}
  line(leftX+200,y+38,midX,y+38); line(midX+200,y+38,rightX,y+38);
  node(leftX,y,status.deviceId||'Presense','simulated source'); node(midX,y,status.protocol||'Protocol','generator '+(status.generatorMode||'unknown')); node(rightX,y,'Endpoint','port '+(status.port||status.uiPort||''));
}
function renderHealthDetails(){
  const el=document.getElementById('healthDetails'); if(!el)return;
  el.innerHTML=[['Protocol endpoint','ready',true],['Generator',status.generatorMode||'unknown',true],['Protocol',status.protocol||'unknown',true],['Values exposed',String(Object.keys((vals.metrics||{})).length),true]].map(x=>'<div class="health-item"><span><span class="dot" style="color:'+(x[2]?'var(--green)':'var(--red)')+'"></span>'+esc(x[0])+'</span><span>'+esc(x[1])+'</span></div>').join('');
}
function metricLabel(key,index){return 'Register '+index+' / '+key}
function render(){cards.innerHTML='<div class="overview-grid">'+[["Device",status.deviceId,"Simulator identity"],["Protocol",status.protocol,"Protocol exposed by Presense"],["Generator",status.generatorMode,"Data shape"],["Port",String(status.port||status.uiPort||""),"Protocol port"]].map(c=>'<div class="overview-card"><div class="k">'+c[2]+'</div><div class="overview-value">'+(c[1]||'unknown')+'</div><div class="overview-hint">'+c[0]+'</div></div>').join("")+'</div>';renderHealthDetails();renderPresenseGraph();values.textContent=JSON.stringify(vals,null,2);const m=(vals.metrics||{});metricBars.innerHTML=Object.entries(m).map(([k,v],idx)=>'<div style="margin:14px 0"><b>'+esc(metricLabel(k,idx))+'</b><div class="k">'+esc(v)+'</div><div class="bar"><span style="width:'+pct(v)+'%"></span></div></div>').join("");components.innerHTML='<div class="grid"><div class="card"><div class="k">Protocol endpoint</div><div class="v">'+esc(status.protocol||'unknown')+'</div><div class="k">port '+esc(status.port||'')+'</div></div><div class="card"><div class="k">UI endpoint</div><div class="v">running</div><div class="k">port '+esc(status.uiPort||status.port||'')+'</div></div><div class="card"><div class="k">Raw '+esc('registers')+'</div><div class="v">'+Object.keys(m).length+'</div><div class="k">current exposed values</div></div></div>';logsData[1].message='Generator mode: '+(status.generatorMode||'unknown');renderLogs()}
async function load(){try{status=await j("/api/status");vals=await j("/api/values");render();if((location.hash.replace("#","")||"dashboard")==="config")loadConfigPage()}catch(e){logsData.push({level:'ERROR',message:e.message});renderLogs()}}
document.addEventListener('change',e=>{if(e.target.classList.contains('log-filter'))renderLogs()});
applyConfigButton.onclick=()=>{configState.textContent="Configuration apply is prepared, but backend write-back is intentionally guarded for this module.";applyConfigButton.disabled=true;};window.addEventListener("hashchange",route);setInterval(load,2000);route();load()
</script></body></html>`))
	})

	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"deviceId":            deviceID,
			"host":                deviceID,
			"protocol":            protocol,
			"port":                port,
			"uiPort":              uiPort,
			"generatorMode":       generatorMode,
			"uptimeSeconds":       int(time.Since(startedAt).Seconds()),
			"supportedGenerators": []string{"sine", "random-int", "random-string", "array", "sawtooth", "square", "triangle"},
		})
	})
	mux.HandleFunc("/api/values", func(w http.ResponseWriter, r *http.Request) {
		values, _ := bank.read(0, 6)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"registers": values,
			"metrics": map[string]any{
				"temperature": float64(values[0]) * 0.1,
				"humidity":    float64(values[1]) * 0.1,
				"pressure":    float64(values[2]) * 0.01,
				"vibration":   float64(values[3]) * 0.01,
				"status":      values[4],
				"cycle":       values[5],
			},
		})
	})

	go func() {
		log.Printf("IoT Presense UI for %s listening on :%d", deviceID, uiPort)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", uiPort), mux); err != nil {
			log.Printf("presense ui stopped: %v", err)
		}
	}()
}

func main() {
	rand.Seed(time.Now().UnixNano())
	deviceID := envString("DEVICE_ID", "machine-01")
	port := envInt("MODBUS_PORT", 5020)
	uiPort := envInt("PRESENSE_UI_PORT", 8300)
	generatorMode := envString("GENERATOR_MODE", "sine")
	tempBase := envFloat("TEMP_BASE", 23.0)
	humidityBase := envFloat("HUMIDITY_BASE", 50.0)
	pressureBase := envFloat("PRESSURE_BASE", 1.2)
	vibrationBase := envFloat("VIBRATION_BASE", 0.5)
	bank := &RegisterBank{}
	startedAt := time.Now()
	go updateRegisters(bank, tempBase, humidityBase, pressureBase, vibrationBase, generatorMode)
	startUI(deviceID, "modbus-tcp", port, uiPort, generatorMode, startedAt, bank)
	address := fmt.Sprintf("0.0.0.0:%d", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", address, err)
	}
	log.Printf("%s Modbus TCP simulator listening on %s", deviceID, address)
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("accept failed: %v", err)
			continue
		}
		go handleConnection(conn, bank)
	}
}
