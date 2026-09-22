package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type NodeValue struct {
	NodeID  string `json:"nodeId"`
	Value   any    `json:"value"`
	Quality string `json:"quality"`
	Unit    string `json:"unit,omitempty"`
}

func getenv(name string, fallback string) string {
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

func randomState(startedAt time.Time) string {
	states := []string{"idle", "running", "warming-up", "blocked", "maintenance", "cooldown"}
	index := int(time.Since(startedAt).Seconds()/3) % len(states)
	if index < 0 {
		index = 0
	}
	return states[index]
}

func generatedNumber(mode string, elapsed float64, base float64, index int) float64 {
	switch mode {
	case "constant":
		return base
	case "random-int":
		return float64(int(base + rand.Float64()*100))
	case "sawtooth":
		return base + math.Mod(elapsed*float64(index+1), 90)
	case "square":
		if int(elapsed/5)%2 == 0 {
			return base + 20
		}
		return base - 20
	case "triangle":
		period := 36.0
		phase := math.Mod(elapsed, period) / period
		if phase < 0.5 {
			return base + phase*40
		}
		return base + (1-phase)*40
	case "random-string":
		return float64(len([]string{"idle", "running", "blocked", "maintenance"}[rand.Intn(4)]))
	case "array":
		return base + float64(index)*10 + rand.Float64()*7
	default:
		return base + math.Sin(elapsed/(9.0+float64(index)*2))*10.0 + rand.Float64()*0.4 - 0.2
	}
}

func nodeValue(nodeID string, startedAt time.Time, tempBase float64, speedBase float64, currentBase float64, generatorMode string) NodeValue {
	if settings := currentSimulatorSettings(); settings != nil {
		tempBase, speedBase, currentBase, generatorMode = settings.Temperature, settings.Speed, settings.Current, settings.GeneratorMode
	}
	elapsed := time.Since(startedAt).Seconds()
	normalized := strings.ToLower(nodeID)
	noise := rand.Float64()*0.4 - 0.2
	if generatorMode == "constant" {
		noise = 0
	}

	switch {
	case strings.Contains(normalized, "temperature"):
		return NodeValue{NodeID: nodeID, Value: generatedNumber(generatorMode, elapsed, tempBase, 0) + noise, Quality: "good", Unit: "°C"}
	case strings.Contains(normalized, "speed"):
		return NodeValue{NodeID: nodeID, Value: generatedNumber(generatorMode, elapsed, speedBase, 1) + noise*10.0, Quality: "good", Unit: "rpm"}
	case strings.Contains(normalized, "current"):
		return NodeValue{NodeID: nodeID, Value: math.Abs(generatedNumber(generatorMode, elapsed, currentBase, 2)/10.0) + noise, Quality: "good", Unit: "A"}
	case strings.Contains(normalized, "load"):
		return NodeValue{NodeID: nodeID, Value: generatedNumber(generatorMode, elapsed, 55.0, 3) + noise*5.0, Quality: "good", Unit: "%"}
	case strings.Contains(normalized, "state") || strings.Contains(normalized, "mode"):
		return NodeValue{NodeID: nodeID, Value: randomState(startedAt), Quality: "good"}
	default:
		return NodeValue{NodeID: nodeID, Value: generatedNumber(generatorMode, elapsed, 1.0, 4) + noise, Quality: "uncertain"}
	}
}

func defaultNodeIDs() []string {
	return []string{
		"ns=2;s=Machine.Temperature",
		"ns=2;s=Machine.Speed",
		"ns=2;s=Machine.Current",
		"ns=2;s=Machine.Load",
		"ns=2;s=Machine.State",
	}
}

func parseNodeIDs(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return defaultNodeIDs()
	}
	parts := strings.Split(raw, ",")
	ids := make([]string, 0, len(parts))
	for _, part := range parts {
		id := strings.TrimSpace(part)
		if id != "" {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return defaultNodeIDs()
	}
	return ids
}

func subscriptionPayload(nodeIDs []string, startedAt time.Time, tempBase float64, speedBase float64, currentBase float64, generatorMode string) map[string]any {
	values := make([]NodeValue, 0, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		values = append(values, nodeValue(nodeID, startedAt, tempBase, speedBase, currentBase, generatorMode))
	}
	return map[string]any{"timestamp": time.Now().UTC().Format(time.RFC3339Nano), "items": values}
}

func subscriptionInterval(raw string) time.Duration {
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		parsed = 1000
	}
	if parsed < 250 {
		parsed = 250
	}
	return time.Duration(parsed) * time.Millisecond
}

func main() {
	rand.Seed(time.Now().UnixNano())
	deviceID := getenv("DEVICE_ID", "opcua-machine-01")
	port := envInt("OPCUA_HTTP_PORT", 4840)
	tempBase := envFloat("TEMP_BASE", 31.0)
	speedBase := envFloat("SPEED_BASE", 1450.0)
	currentBase := envFloat("CURRENT_BASE", 8.5)
	generatorMode := getenv("GENERATOR_MODE", "sine")
	startedAt := time.Now()

	mux := http.NewServeMux()
	initSimulatorSettings(simulatorSettings{Protocol: "opcua", GeneratorMode: generatorMode, Temperature: tempBase, Speed: speedBase, Current: currentBase})
	registerSimulatorConfig(mux, deviceID, getenv("PRESENSE_HOST", deviceID), port)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html><html><head><meta charset="utf-8"><title>IoT Presense</title><style>
:root{--bg:#14161a;--side:#101216;--panel:#1b1f26;--border:#2a2f38;--text:#e7e9ee;--muted:#8a9099;--accent:#8a9099;--green:#28c76f;--yellow:#febe3b;--red:#ff4d4f}
*{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--text);font-family:Inter,system-ui,Segoe UI,Arial,sans-serif}.layout{display:grid;grid-template-columns:250px 1fr;min-height:100vh}aside{background:var(--side);border-right:1px solid var(--border);padding:18px}.brand{font-size:22px;font-weight:800;margin-bottom:22px}.brand span{color:var(--accent)}a{display:block;color:var(--text);text-decoration:none;padding:11px 12px;border-radius:10px;margin:6px 0;border:1px solid transparent}a.active,a:hover{background:var(--panel);border-color:var(--border)}main{padding:22px}.card{background:var(--panel);border:1px solid var(--border);border-radius:16px;padding:16px;margin-bottom:14px}.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(260px,1fr));gap:14px}.k{color:var(--muted);font-size:12px}.v{font-size:24px;font-weight:800;margin-top:6px}pre{background:#0e0f12;border:1px solid var(--border);border-radius:12px;padding:12px;overflow:auto}.hidden{display:none}.bar{height:10px;background:#0e0f12;border:1px solid var(--border);border-radius:999px;overflow:hidden}.bar span{display:block;height:100%;background:var(--accent)}.toolbar{display:flex;gap:10px;flex-wrap:wrap;margin-bottom:14px}label{display:inline-flex;align-items:center;gap:6px;background:#0e0f12;border:1px solid var(--border);border-radius:999px;padding:8px 11px}.log-row{background:#0e0f12;border:1px solid var(--border);border-radius:14px;padding:12px;margin:10px 0;font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace}.log-info{color:var(--green)}.log-warn{color:var(--yellow)}.log-error{color:var(--red)}.dot{display:inline-block;width:10px;height:10px;border-radius:50%;margin-right:8px;background:currentColor}.overview-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:12px;margin-bottom:14px}.overview-card{background:#11151b;border:1px solid var(--border);border-radius:14px;padding:12px}.overview-value{font-size:20px;font-weight:800;margin-top:6px}.overview-hint{color:var(--muted);font-size:12px;margin-top:5px}.health-layout{display:grid;grid-template-columns:360px 1fr;gap:14px;margin-bottom:14px}.health-pie{width:190px;height:190px;border-radius:50%;background:var(--green);margin:22px auto 10px}.health-item{display:flex;justify-content:space-between;border-bottom:1px solid var(--border);padding:8px 0}.connection-graph{position:relative;min-height:260px;height:260px;overflow:hidden;background:#11151b;border:1px solid var(--border);border-radius:16px;margin-top:12px}.edge-svg{position:absolute;left:0;top:0;width:100%;height:100%;pointer-events:none}.node{position:absolute;min-width:150px;border:1px solid var(--accent);border-radius:14px;background:var(--panel);padding:10px 12px;box-shadow:0 8px 20px rgba(0,0,0,.25);z-index:2}.node .title{font-weight:800;font-size:15px}.node .sub{color:var(--muted);font-size:12px;margin-top:4px}
</style></head><body><div class="layout"><aside><div class="brand">IoT <span>Presense</span></div><a href="#dashboard">Dashboard</a><a href="#metrics">Quick Metrics</a><a href="#components">Components</a><a href="#config">Configuration</a><a href="#logs">Logs</a></aside><main>
<section id="page-dashboard"><div id="cards"></div><div class="health-layout"><div class="card"><h3>Health distribution</h3><div class="health-pie"></div><div class="k" style="text-align:center">healthy: 1<br>unhealthy: 0</div></div><div class="card"><h3>Health details</h3><div id="healthDetails"></div></div></div><div class="card"><h3>Connection graph</h3><div id="connectionGraph" class="connection-graph"></div></div><div class="card"><h3>Current values</h3><pre id="values"></pre></div></section>
<section id="page-metrics" class="hidden"><div class="card"><h3>Quick Metrics</h3><div id="metricBars"></div></div></section>
<section id="page-components" class="hidden"><div class="card"><h3>Components</h3><div id="components"></div></div></section>
<section id="page-config" class="hidden"><div class="card"><h3>Configuration</h3><div class="toolbar"><button onclick="loadConfigPage()">Refresh configuration</button><button onclick="validateConfigDraft()">Validate changes</button><button id="applyConfigButton" disabled>Apply proposal</button><button onclick="resetConfigDraft()">Reset editor</button><span id="configState" class="k">Idle</span></div><textarea id="configEditor" spellcheck="false" style="width:100%;min-height:360px;background:#0e0f12;color:var(--text);border:1px solid var(--border);border-radius:12px;padding:12px;font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace"></textarea><h3>Validation and diff</h3><pre id="configDiff"></pre><h3>Persistence</h3><div class="k">Settings are saved when PRESENSE_CONFIG_PATH is configured.</div></div></section>
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
function route(){const p=location.hash.replace("#","")||"dashboard";["dashboard","metrics","components","config","logs"].forEach(x=>document.getElementById("page-"+x).classList.toggle("hidden",x!==p));document.querySelectorAll("aside a").forEach(a=>a.classList.toggle("active",a.getAttribute("href")==="#"+p));render()}
let configOriginal='';
function defaultNodeConfig(){
  const items=vals.items||[];
  if(items.length){
    return items.map((item,index)=>({name:String(item.nodeId||('node-'+index)).split('.').pop().toLowerCase(),nodeId:item.nodeId,dataType:typeof item.value==='string'?'string':'float64',generatorMode:typeof item.value==='string'?'random-string':(status.generatorMode||'sine'),quality:item.quality||'good'}));
  }
  return [
    {name:'temperature',nodeId:'ns=2;s=Machine.Temperature',dataType:'float64',generatorMode:status.generatorMode||'sine'},
    {name:'speed',nodeId:'ns=2;s=Machine.Speed',dataType:'float64',generatorMode:status.generatorMode||'sine'},
    {name:'current',nodeId:'ns=2;s=Machine.Current',dataType:'float64',generatorMode:status.generatorMode||'sine'},
    {name:'load',nodeId:'ns=2;s=Machine.Load',dataType:'float64',generatorMode:status.generatorMode||'sine'},
    {name:'state',nodeId:'ns=2;s=Machine.State',dataType:'string',generatorMode:'random-string'}
  ];
}
function currentConfig(){
  return {
    deviceId:status.deviceId||'unknown',
    generatorMode:status.generatorMode||'sine',
    host:status.host||status.deviceId||'localhost',
    port:Number(status.port||4840),
    protocol:status.protocol||'opcua',
    uiPort:Number(status.uiPort||status.port||0),
    supportedGenerators:status.supportedGenerators||['sine','random-int','random-string','array','sawtooth','square','triangle'],
    nodes:defaultNodeConfig(),
    currentValues:vals
  };
}
function pct(v){return Math.max(0,Math.min(100,Number(v)||0))}
function logClass(level){return level==='ERROR'?'log-error':level==='WARN'?'log-warn':'log-info'}
function renderLogs(){const selected=new Set(Array.from(document.querySelectorAll('.log-filter:checked')).map(x=>x.value));logs.innerHTML=logsData.filter(x=>selected.has(x.level)).map(x=>'<div class="log-row"><span class="'+logClass(x.level)+'"><span class="dot"></span>'+esc(x.level)+'</span> '+esc(x.message)+'</div>').join('')}

function renderPresenseGraph(){
  const box=document.getElementById('connectionGraph'); if(!box)return;
  const w=1040, h=260; box.style.width=w+"px"; if(window.IoTWorkspace)IoTWorkspace.graphViewport(box); box.style.height=h+'px';
  const leftX=40, midX=Math.round(w/2)-110, rightX=w-230, y=95;
  box.innerHTML='<svg class="edge-svg" width="'+w+'" height="'+h+'" viewBox="0 0 '+w+' '+h+'"></svg>';
  const svg=box.querySelector('svg');
  function line(x1,y1,x2,y2){const p=document.createElementNS('http://www.w3.org/2000/svg','path');const dx=Math.max(80,Math.abs(x2-x1)*0.45);p.setAttribute('d','M '+x1+' '+y1+' C '+(x1+dx)+' '+y1+', '+(x2-dx)+' '+y2+', '+x2+' '+y2);p.setAttribute('stroke','#8a9099');p.setAttribute('stroke-width','3');p.setAttribute('fill','none');svg.appendChild(p);}
  function node(x,y,title,sub){const n=document.createElement('div');n.className='node';n.style.left=x+'px';n.style.top=y+'px';n.style.width='200px';n.innerHTML='<div class="title">'+esc(title)+'</div><div class="sub">'+esc(sub||'')+'</div>';box.appendChild(n);}
  line(leftX+200,y+42,midX,y+42); line(midX+200,y+42,rightX,y+42);
  node(leftX,y,status.deviceId||'Presense','simulated source'); node(midX,y,status.protocol||'Protocol','generator '+(status.generatorMode||'unknown')); node(rightX,y,'Endpoint','port '+(status.port||status.uiPort||''));
}
function renderHealthDetails(){
  const el=document.getElementById('healthDetails'); if(!el)return;
  el.innerHTML=[['Protocol endpoint','ready',true],['Generator',status.generatorMode||'unknown',true],['Protocol',status.protocol||'unknown',true],['Values exposed',String(Object.keys((vals.metrics||{})).length),true]].map(x=>'<div class="health-item"><span><span class="dot" style="color:'+(x[2]?'var(--green)':'var(--red)')+'"></span>'+esc(x[0])+'</span><span>'+esc(x[1])+'</span></div>').join('');
}
function metricLabel(key,index){return 'Node '+index+' / '+key}
function render(){cards.innerHTML='<div class="overview-grid">'+[["Device",status.deviceId,"Simulator identity"],["Protocol",status.protocol,"Protocol exposed by Presense"],["Generator",status.generatorMode,"Data shape"],["Port",String(status.port||status.uiPort||""),"Protocol port"]].map(c=>'<div class="overview-card"><div class="k">'+c[2]+'</div><div class="overview-value">'+(c[1]||'unknown')+'</div><div class="overview-hint">'+c[0]+'</div></div>').join("")+'</div>';renderHealthDetails();renderPresenseGraph();values.textContent=JSON.stringify(vals,null,2);const m=Object.fromEntries((vals.items||[]).map(x=>[x.nodeId,x.value]));metricBars.innerHTML=Object.entries(m).map(([k,v],idx)=>'<div style="margin:14px 0"><b>'+esc(metricLabel(k,idx))+'</b><div class="k">'+esc(v)+'</div><div class="bar"><span style="width:'+pct(v)+'%"></span></div></div>').join("");components.innerHTML='<div class="grid"><div class="card"><div class="k">Protocol endpoint</div><div class="v">'+esc(status.protocol||'unknown')+'</div><div class="k">port '+esc(status.port||'')+'</div></div><div class="card"><div class="k">UI endpoint</div><div class="v">running</div><div class="k">port '+esc(status.uiPort||status.port||'')+'</div></div><div class="card"><div class="k">Raw '+esc('nodes')+'</div><div class="v">'+Object.keys(m).length+'</div><div class="k">current exposed values</div></div></div>';logsData[1].message='Generator mode: '+(status.generatorMode||'unknown');renderLogs()}
async function load(){try{status=await j("/health");vals=await j("/api/values");render();}catch(e){logsData.push({level:'ERROR',message:e.message});renderLogs()}}
document.addEventListener('change',e=>{if(e.target.classList.contains('log-filter'))renderLogs()});
window.addEventListener("hashchange",route);setInterval(load,2000);route();load()
</script><script src="/static/workspace-ui.js"></script><script src="/static/config-form.js"></script><script src="/static/presense-ui.js"></script><script src="/static/lab-link.js"></script></body></html>`))
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"deviceId":            deviceID,
			"host":                deviceID,
			"status":              "running",
			"protocol":            "opcua",
			"port":                port,
			"uiPort":              port,
			"generatorMode":       simulatorMode(generatorMode),
			"supportedGenerators": []string{"sine", "random-int", "random-string", "array", "sawtooth", "square", "triangle"},
			"uptime":              time.Since(startedAt).String(),
		})
	})

	mux.HandleFunc("/api/values", func(w http.ResponseWriter, r *http.Request) {
		nodeIDs := defaultNodeIDs()
		payload := subscriptionPayload(nodeIDs, startedAt, tempBase, speedBase, currentBase, generatorMode)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	})

	mux.HandleFunc("/subscribe", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming is not supported", http.StatusInternalServerError)
			return
		}
		nodeIDs := parseNodeIDs(r.URL.Query().Get("nodeIds"))
		interval := subscriptionInterval(r.URL.Query().Get("intervalMs"))
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		writeEvent := func() bool {
			payload := subscriptionPayload(nodeIDs, startedAt, tempBase, speedBase, currentBase, generatorMode)
			encoded, err := json.Marshal(payload)
			if err != nil {
				return false
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", encoded); err != nil {
				return false
			}
			flusher.Flush()
			return true
		}

		if !writeEvent() {
			return
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-r.Context().Done():
				return
			case <-ticker.C:
				if !writeEvent() {
					return
				}
			}
		}
	})

	mux.HandleFunc("/read", func(w http.ResponseWriter, r *http.Request) {
		nodeID := r.URL.Query().Get("nodeId")
		if nodeID == "" {
			http.Error(w, "nodeId is required", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(nodeValue(nodeID, startedAt, tempBase, speedBase, currentBase, generatorMode))
	})

	addr := fmt.Sprintf("0.0.0.0:%d", port)
	log.Printf("%s OPC UA demo simulator listening on %s", deviceID, addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
