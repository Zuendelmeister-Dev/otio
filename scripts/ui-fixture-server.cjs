// Browser regression fixtures only. Not a replacement for the running demo stack.
// node scripts/ui-fixture-server.cjs -> Lense :18000, Sense :18100
const http=require('node:http'),fs=require('node:fs'),path=require('node:path');
const root=path.resolve(__dirname,'..');
const load=p=>JSON.parse(fs.readFileSync(path.join(root,p),'utf8'));
const topology=load('apps/lense/config/topology.json');
const config=load('apps/sense/config-modbus/config.json');
for(const [app,port]of [['lense',18000],['sense',18100]])http.createServer((req,res)=>{
 const url=new URL(req.url,'http://localhost');
 if(url.pathname.startsWith('/protocol-lab/')){const proxy=http.request({hostname:'127.0.0.1',port:18500,path:req.url.replace('/protocol-lab',''),method:req.method,headers:req.headers},upstream=>{res.writeHead(upstream.statusCode,upstream.headers);upstream.pipe(res);});proxy.on('error',()=>{res.statusCode=503;res.end('{"error":"Test gateway unavailable"}');});req.pipe(proxy);return;}
 if(url.pathname==='/protocols'){res.writeHead(302,{Location:'/protocol-lab/static/'});res.end();return;}
 const now=new Date().toISOString();
 const componentStatus=Object.fromEntries([...topology.senses,...topology.dispenses,...topology.targets,{id:'iot-lense'},{id:'mqtt'},{id:'postgres'}].map(x=>[x.id,{connected:true,healthy:true,lastSeen:now}]));
 const agents=topology.agents.map(a=>({...a,connected:true,healthy:true,timestamp:now}));
 const points=Array.from({length:30},(_,i)=>({timestamp:new Date(Date.now()-(29-i)*2000).toISOString(),value:40+Math.sin(i)*3}));
 const routes={
 '/api/live-messages':{label:url.searchParams.get('id'),scope:'Fixture: successful publications',sampledAt:now,unavailable:[],items:[{timestamp:now,topic:'fixture/value',payload:'<test value>',truncated:false}]},
  '/api/graph-traffic':{items:Object.fromEntries([...topology.senses,...topology.dispenses,...topology.targets,{id:'mqtt'}].map(x=>[x.id,{available:true,avgPerMinute:36.2,sampledAt:now,scope:'UI fixture: successful publications',last:{timestamp:now,topic:'iot-lense/modbus-machine-01/metrics/temperature',payload:JSON.stringify({value:23.5,unit:'°C'})}}]))},
  '/api/database/tables':{database:'iotdb (UI fixture)',tables:{metric_events:[{name:'ts',type:'timestamp with time zone'},{name:'agent_id',type:'text'},{name:'metric_value',type:'double precision'}]}},
  '/api/database/query':{columns:['id','ts','agent_id','topic','metric_name','metric_value','metric_text','metric_unit','metric_type','source_type','source_host','source_address','quality_status','payload'],rows:Array.from({length:25},(_,i)=>[i+1,now,'modbus-machine-01','iot-lense/modbus-machine-01/metrics/temperature','temperature',23.5,null,'°C','gauge','modbus-tcp','presense-modbus-01','holding-register:0','good',JSON.stringify({agentId:'modbus-machine-01',metric:{name:'temperature',value:23.5},source:{host:'presense-modbus-01',address:'holding-register:0'},literal:'<test value>'})]),limit:200,truncated:false},
  '/api/logs':{items:[{level:'ERROR',component:'opcua-machine-01',message:'unexpected EOF',timestamp:now}]},
  '/api/topology':topology,
  '/api/agents/iot-sense-modbus-01':{agent:{agentId:'iot-sense-modbus-01',sourceType:'sense',connected:true,healthy:true},topics:['iot-lense/modbus-machine-01/metrics/temperature'],messages:Array.from({length:50},(_,i)=>({timestamp:now,topic:'iot-lense/modbus-machine-01/metrics/temperature',payload:{value:23.5,index:i,literal:'<test value>'}}))},
  '/api/agents/modbus-machine-01':{agent:agents[0],topics:['iot-lense/modbus-machine-01/metrics/temperature'],messages:[]},
  '/api/uns':{tree:{'iot-lense':{'modbus-machine-01':{metrics:{temperature:{}},errors:{},status:{}}}}},
  '/api/runtime':{mqtt:{host:'mqtt',port:1883,topicPrefix:'iot-lense'},postgres:{host:'postgres',port:5432,database:'iotdb',user:'iot'},topology,componentStatus},
  '/api/summary':{mqtt:{connected:true,lastMessage:now,messageCount:6523},connectedAgents:5,health:{healthy:5,unhealthy:0},messagesPerAgent5m:{avg:5,min:4,max:6},latestErrors:[],componentStatus},
  '/api/agents':{items:agents},'/api/catalog':{agents:agents.map(a=>a.agentId),metrics:['temperature','humidity']},
  '/api/status':{broker:{connected:true,lastPublish:now,publishCount:500},config:{...config.broker,valid:true,loaded:true,brokerHost:'mqtt',brokerPort:1883},sources:Object.fromEntries(config.sources.map(s=>[s.agentId,{connected:true,healthy:true,lastRead:now,lastReadAgoHuman:'0s ago'}]))},
  '/api/metrics':Object.fromEntries(config.sources.map(s=>[s.agentId,{humidity:points,temperature:points}])),
  '/api/config':{raw:JSON.stringify(config,null,2),parsed:config},'/api/config/history':{items:[]},
 };
 if(url.pathname.startsWith('/api/')){res.setHeader('Access-Control-Allow-Origin','*');res.setHeader('Content-Type','application/json');res.end(JSON.stringify(routes[url.pathname]||{items:[]}));return;}
 let file=path.join(root,'apps',app,'static','index.html');
 if(url.pathname.startsWith('/static/')){const name=path.basename(url.pathname);file=path.join(root,'shared/web',name);if(!fs.existsSync(file))file=path.join(root,'apps',app,'static',name);}
 try{res.setHeader('Content-Type',file.endsWith('.js')?'text/javascript':'text/html');res.end(fs.readFileSync(file));}catch{res.statusCode=404;res.end('not found');}
}).listen(port,'127.0.0.1',()=>console.log(`${app} UI fixture: http://127.0.0.1:${port}`));
