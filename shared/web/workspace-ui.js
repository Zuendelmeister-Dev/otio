(function (root) {
  'use strict';
  function statusDomain(points, now) {
    const times = points.map(p => Date.parse(p.timestamp)).filter(Number.isFinite);
    const first = times.length ? Math.min(...times) : now;
    return {start: Math.min(first - 1000, now - 30000), end: now + 1000};
  }
  function graphViewport(box) {
    if (!box) return;
    let view = box.parentElement;
    if (!view.classList.contains('graph-scroll')) {
      view = document.createElement('div'); view.className = 'graph-scroll';
      box.before(view); view.append(box);
    }
    if (view.dataset.workspacePan) return;
    view.dataset.workspacePan = 'true'; view.tabIndex = 0;
    view.setAttribute('aria-label', 'Connection graph. Scroll or drag to pan; use arrow keys when focused.');
    let drag = null, moved = false;
    view.addEventListener('pointerdown', e => {
      if (e.button !== 0 || e.pointerType === 'touch' || e.target.closest('button,input,select,a')) return;
      delete view.dataset.dragged;
      drag = {x:e.clientX, y:e.clientY, left:view.scrollLeft, top:view.scrollTop}; moved = false;
    });
    view.addEventListener('pointermove', e => {
      if (!drag) return;
      if (!(e.buttons & 1)) {drag=null;view.classList.remove('panning');return;}
      const dx=e.clientX-drag.x, dy=e.clientY-drag.y;
      if (Math.abs(dx)+Math.abs(dy)>5) {moved=true; view.setPointerCapture(e.pointerId); view.classList.add('panning');}
      if (moved) {view.dataset.dragged='true';view.scrollLeft=drag.left-dx; view.scrollTop=drag.top-dy; e.preventDefault();}
    });
    const stop=()=>{drag=null;view.classList.remove('panning');};
    view.addEventListener('pointerup',stop); view.addEventListener('pointercancel',stop);
    view.addEventListener('click',e=>{if(moved){e.preventDefault();e.stopImmediatePropagation();moved=false;}},true);
    view.addEventListener('keydown',e=>{if(e.target!==view)return;const shifts={ArrowLeft:-120,ArrowRight:120};if(e.key in shifts){view.scrollLeft+=shifts[e.key];e.preventDefault();}});
  }
  function graphLanes(box, lanes) {
    graphViewport(box);
    box.querySelectorAll('.graph-lane').forEach(el=>el.remove());
    lanes.forEach(({x,width,label})=>{
      const lane=document.createElement('div');lane.className='graph-lane';
      lane.style.left=x+'px';lane.style.width=width+'px';lane.textContent=label;box.prepend(lane);
    });
  }
  const palette={presense:'#8a9099',sense:'#14b8a6',lense:'#ff8a1d',dispense:'#ef4444',system:'#facc15'};
  function isSimulation(source){return /presense/i.test([source.origin,source.managedBy,source.sourceHost,source.host,source.kind].filter(Boolean).join(' '));}
  function topologyModel(t={}){
    const senses=t.senses||[],dispenses=t.dispenses||[],sources=t.presenses?.length?t.presenses:t.agents||[];
    return [...sources.map(s=>({...s,id:s.agentId||s.id,label:s.agentId||s.id,kind:'presense',simulation:isSimulation(s),flows:[s.senseId||senses.find(x=>x.sourceType===s.sourceType)?.id].filter(Boolean)})),
      ...senses.map(s=>({...s,kind:'sense',flows:[s.id]})),
      ...dispenses.map(d=>({...d,kind:'dispense',flows:senses.filter(s=>s.sourceType===d.sourceType).map(s=>s.id)})),
      ...(t.targets||[]).map(t=>({...t,kind:'system',flows:senses.filter(s=>dispenses.some(d=>d.target===t.id&&d.sourceType===s.sourceType)).map(s=>s.id)})),
      {id:'mqtt',label:'MQTT Broker',kind:'system',flows:[...senses.map(s=>s.id),'historian']},
      {id:'iot-lense',label:'IoT Lense',kind:'lense',flows:['historian']},{id:'postgres',label:'Postgres',kind:'system',flows:['historian']}];
  }
  let activeTooltip;
  function hideTooltip(){if(activeTooltip)activeTooltip.hidden=true;}
  function showTooltip(target,text,event,above=false){
    if(!text){hideTooltip();return;}
    if(!activeTooltip){activeTooltip=document.createElement('div');activeTooltip.id='workspace-tooltip';activeTooltip.setAttribute('role','tooltip');activeTooltip.style.cssText='position:fixed;z-index:10000;max-width:320px;padding:9px 12px;background:#090e16;color:#e6edf5;border:1px solid #526277;border-radius:8px;font:13px/1.5 system-ui;white-space:pre-wrap;overflow-wrap:anywhere;max-height:260px;overflow:hidden;box-shadow:0 4px 16px #0006;pointer-events:none';document.body.append(activeTooltip);}
    activeTooltip.textContent=text;activeTooltip.hidden=false;
    const bounds=target.getBoundingClientRect(),x=event?.clientX??bounds.left,y=event?.clientY??bounds.bottom;
    activeTooltip.style.left=Math.max(8,Math.min(x+14,window.innerWidth-activeTooltip.offsetWidth-8))+'px';
    activeTooltip.style.top=Math.max(8,Math.min(above?bounds.top-activeTooltip.offsetHeight-8:y+14,window.innerHeight-activeTooltip.offsetHeight-8))+'px';
  }
  function healthSegmentAt(x,y,count){
    if(!count||Math.hypot(x-85,y-42)>34)return -1;
    const angle=(Math.atan2(y-42,x-85)+Math.PI/2+2*Math.PI)%(2*Math.PI);
    return Math.min(count-1,Math.floor(angle/(2*Math.PI)*count));
  }
  function topicLogRef(topic){const parts=topic.split('/'),kind=parts.at(-1);return parts.length>=3&&['errors','logs'].includes(kind)?{agentId:parts.at(-2),kind}:null;}
  function topicLogs(items,ref){return (items||[]).filter(item=>(item.component===ref.agentId||item.agentId===ref.agentId)&&(ref.kind==='logs'||['ERROR','WARN','WARNING'].includes(String(item.level||item.severity||'').toUpperCase())));}
  function drawHealthPie(canvas,healthy,unhealthy){
    if(!canvas)return;const details=canvas.closest('.health-compact')?.querySelector('[id$="HealthDetails"],#healthDetails');
    const rows=[...(details?.querySelectorAll('.health-item')||[])];
    const segments=rows.length?rows.map(row=>({label:row.title||[...row.children].map(el=>el.textContent.trim()).join(': '),ok:!!row.querySelector('.dot.ok')||!!row.querySelector('.dot[style*="green"]')})):[{label:healthy+' healthy',ok:true},{label:unhealthy+' unhealthy',ok:false}].filter((_,i)=>i?unhealthy:healthy);
    const ratio=window.devicePixelRatio||1,w=170,h=110;canvas.width=w*ratio;canvas.height=h*ratio;const ctx=canvas.getContext('2d');ctx.scale(ratio,ratio);
    segments.forEach((item,i)=>{const start=-Math.PI/2+i*2*Math.PI/segments.length,end=start+2*Math.PI/segments.length;ctx.beginPath();ctx.moveTo(85,42);ctx.arc(85,42,34,start+.035,end-.035);ctx.closePath();ctx.fillStyle=item.ok?'#28c76f':'#ff4d4f';ctx.fill();});
    ctx.font='11px system-ui';ctx.fillStyle='#d9e6f5';ctx.textAlign='center';ctx.fillText(segments.filter(s=>s.ok).length+' / '+segments.length+' checks healthy',85,98);canvas.setAttribute('aria-label',segments.map(s=>s.label).join('; '));
    canvas.tabIndex=0;canvas.removeAttribute('title');canvas.setAttribute('aria-describedby','workspace-tooltip');
    canvas.onpointermove=event=>{const r=canvas.getBoundingClientRect(),i=healthSegmentAt((event.clientX-r.left)*w/r.width,(event.clientY-r.top)*h/r.height,segments.length);showTooltip(canvas,segments[i]?.label,event);};
    canvas.onpointerleave=hideTooltip;canvas.onblur=hideTooltip;canvas.onfocus=()=>showTooltip(canvas,segments.map(s=>s.label).join(' · '));
  }
  function explainError(message){if(/unexpected EOF|\bEOF\b/i.test(message))return 'The connection ended before a complete response arrived. Possible causes: device restart, network interruption, or the wrong protocol/port. Check device logs, endpoint URL and protocol. EOF alone does not identify the cause.';if(/timeout|deadline exceeded/i.test(message))return 'No complete response arrived within the allowed time. Check reachability, firewall rules and device load.';return '';}
  function componentRoute(id,configure=false){return !configure&&id==='iot-lense'?'dashboard':(configure?'configurations/':'agents/')+encodeURIComponent(id||'');}
  function relatedFlows(a,b){return a.some(flow=>b.includes(flow));}
  let navigationNodes=[];
  let graphTraffic={};
  function hasLiveMessages(info){return ['sense','dispense'].includes(info.kind)||info.id==='mqtt'||(info.kind==='system'&&info.flows?.length&&info.id!=='postgres');}
  function trafficLabel(sample){return sample?.available?'Ø '+Number(sample.avgPerMinute).toFixed(1)+' msg/min · 5m':'Ø — msg/min · 5m';}
  async function refreshGraphTraffic(){try{const url=lenseURL();url.pathname='/api/graph-traffic';const response=await fetch(url,{cache:'no-store',signal:AbortSignal.timeout(6000)});if(!response.ok)throw Error();const data=await response.json();graphTraffic=data.items||{};}catch{graphTraffic={};}decorateGraphs();}

  function lenseURL(){const explicit=new URLSearchParams(location.search).get('lense');let u;try{u=new URL(explicit||location.href);if(!['http:','https:'].includes(u.protocol))throw Error();}catch{u=new URL(location.href);}if(!explicit&&!document.title.includes('Lense'))u.port='8000';u.hash='';u.search='';u.pathname='/';return u;}
  function navigationURL(raw){if(!raw)return null;try{const u=new URL(raw,location.href);if(!['http:','https:'].includes(u.protocol))return null;if(['localhost','127.0.0.1'].includes(u.hostname))u.hostname=location.hostname;return u;}catch{return null;}}
  function setTopology(t){navigationNodes=topologyModel(t);if(typeof document!=='undefined')decorateGraphs();}
  function nodeInfo(node){const title=node.querySelector('.title')?.textContent;return node.graphInfo||navigationNodes.find(n=>n.id===title||n.label===title||(title==='Input MQTT Broker'&&n.id==='mqtt')||(title==='Target MQTT Broker'&&n.flows?.length&&n.kind==='system'&&n.id!=='mqtt'&&n.id!=='postgres'))||{id:title,label:title,kind:node.classList.contains('sense')?'sense':node.classList.contains('presense')?'presense':'system',flows:[node.parentElement.dataset.defaultFlow||'local']};}
  function decorateGraphs(){document.querySelectorAll('.node').forEach(node=>{const info=nodeInfo(node);node.tabIndex=0;node.setAttribute('role','button');node.setAttribute('aria-label','Details: '+(info.label||info.id));node.classList.toggle('presense-simulation',!!info.simulation);node.style.borderColor=palette[info.kind]||palette.system;if(hasLiveMessages(info)){let badge=node.querySelector('.traffic-rate');if(!badge){badge=document.createElement('div');badge.className='traffic-rate';badge.style.cssText='font-size:11px;color:#bdc9d8;margin-top:4px;font-variant-numeric:tabular-nums';node.append(badge);}const sample=graphTraffic[info.id],label=trafficLabel(sample);if(badge.textContent!==label)badge.textContent=label;}if(!node.dataset.explicitFlows)node.dataset.flows=JSON.stringify(info.flows||['local']);});document.querySelectorAll('path[data-flow-instance]').forEach(edge=>{const info=navigationNodes.find(n=>n.id===edge.dataset.flowInstance);edge.dataset.flows=JSON.stringify(info?.flows||[edge.dataset.flowInstance]);});const hovered=document.querySelector('.node:hover');if(hovered)highlightFlow(hovered);}
  function highlightFlow(node){const box=node.parentElement,selected=JSON.parse(node.dataset.flows||'["local"]');box.querySelectorAll('.node,svg path').forEach(item=>{const flows=JSON.parse(item.dataset.flows||'["local"]');item.classList.toggle('flow-muted',!relatedFlows(selected,flows));});}
  function clearFlow(box){box?.querySelectorAll('.flow-muted').forEach(el=>el.classList.remove('flow-muted'));}
  function renderPresenseGraph(box,status){
    const source=navigationNodes.find(n=>n.id===status.deviceId),flows=source?.flows||['local'];
    const collectors=navigationNodes.filter(n=>n.kind==='sense'&&relatedFlows(flows,n.flows));
    const nodes=[{...(source||{}),id:status.deviceId||'Presense',label:status.deviceId||'Presense',kind:'presense',simulation:true,flows,url:location.href,description:(status.protocol||'Protocol')+' · listening on port '+(status.port||status.uiPort),local:'#config'},...collectors];
    if(collectors.length)nodes.push(navigationNodes.find(n=>n.id==='mqtt'));
    const width=Math.max(980,nodes.length*340);box.style.width=width+'px';box.style.height='240px';box.replaceChildren();
    const svg=document.createElementNS('http://www.w3.org/2000/svg','svg');svg.setAttribute('width',width);svg.setAttribute('height',240);svg.classList.add('edge-svg');box.append(svg);
    nodes.forEach((info,i)=>{const el=document.createElement('div');el.className='node '+info.kind;el.graphInfo=info;el.style.cssText='left:'+(30+i*340)+'px;top:80px;width:260px';const title=document.createElement('div'),sub=document.createElement('div');title.className='title';title.textContent=info.label||info.id;sub.className='sub';sub.textContent=info.description||(info.kind==='sense'?'Configured collector':'Shared message broker');el.append(title,sub);box.append(el);
      if(i){const edge=document.createElementNS(svg.namespaceURI,'path');edge.setAttribute('d','M '+(290+(i-1)*340)+' 122 L '+(30+i*340)+' 122');edge.setAttribute('stroke',palette[nodes[i-1].kind]);edge.setAttribute('stroke-width','3');edge.dataset.flows=JSON.stringify(flows);svg.append(edge);}
    });
    if(!collectors.length){const note=document.createElement('p');note.className='k';note.style.cssText='position:absolute;left:30px;bottom:12px';note.textContent='No collector mapped in Lense topology. Configure a Sense collector to read this simulator.';box.append(note);}
    graphViewport(box);decorateGraphs();
  }
  function mount() {
    document.querySelectorAll('aside .brand').forEach(brand=>{
      const link=document.createElement('a');link.className='brand home-link';link.href='#dashboard';
      link.setAttribute('aria-label','Instance dashboard');link.innerHTML=brand.innerHTML;brand.replaceWith(link);
    });
    document.querySelectorAll('#healthDetails,#senseHealthDetails,#dispenseHealthDetails').forEach(details=>{
      const card=details.closest('.card');if(!card)return;
      const summary=card.parentElement;summary.classList.add('health-compact');const page=summary.closest('section');if(page)page.prepend(summary);
      const legacy=summary.querySelector('.health-pie');if(legacy){legacy.style.cssText='background:none;width:170px;height:110px;margin:0;border-radius:0';legacy.append(document.createElement('canvas'));const caption=legacy.nextElementSibling;if(caption)caption.hidden=true;}
      const redraw=()=>{const canvas=summary.querySelector('canvas');if(canvas)drawHealthPie(canvas,0,0);};new MutationObserver(redraw).observe(details,{childList:true,subtree:true});redraw();
      const heading=card.querySelector('h3');if(heading)heading.textContent='Status';
    });
    const dialog=document.createElement('dialog');dialog.className='node-dialog';dialog.setAttribute('aria-label','Connection details');
    dialog.innerHTML='<form method="dialog"><button class="close-detail" aria-label="Close details">×</button></form><h2></h2><p></p><div class="node-actions"></div>';
    document.body.append(dialog);
    function inspect(node){const info=nodeInfo(node);dialog.querySelector('h2').textContent=info.label||info.id||'Connection';dialog.querySelector('p').textContent=node.querySelector('.sub')?.textContent||'Configured component';const actions=dialog.querySelector('.node-actions');actions.replaceChildren();
      function link(label,url){if(!url)return;const a=document.createElement('a');a.textContent=label;a.href=url;a.onclick=()=>dialog.close();actions.append(a);}
      const central=lenseURL();central.hash=componentRoute(info.id);link('View in Lense',central.href);const config=new URL(central);config.hash=componentRoute(info.id,true);link('Configure in Lense',config.href);if(hasLiveMessages(info)){const live=new URL(central);live.hash='live-messages/'+encodeURIComponent(info.id);link('Live Messages',live.href);}
      const remote=navigationURL(info.url);if(remote){remote.hash='dashboard';remote.searchParams.set('lense',lenseURL().href);link('Open instance UI',remote.href);}if(info.local)link('Configure this instance',info.local);else if((document.title.includes('Sense')&&info.kind==='sense')||(document.title.includes('Dispense')&&info.kind==='dispense'))link('Configure this instance','#config');dialog.showModal();
    }
    document.addEventListener('click',event=>{const node=event.target.closest('.node');const view=event.target.closest('.graph-scroll');if(view?.dataset.dragged==='true'){delete view.dataset.dragged;event.preventDefault();event.stopPropagation();return;}if(!node)return;event.preventDefault();event.stopPropagation();inspect(node);},true);
    document.addEventListener('keydown',event=>{if(event.target.matches('.node')&&['Enter',' '].includes(event.key)){event.preventDefault();inspect(event.target);}});
    document.addEventListener('pointerover',event=>{const node=event.target.closest('.node');if(node)highlightFlow(node);});
    document.addEventListener('pointerout',event=>{const node=event.target.closest('.node');if(node&&!node.contains(event.relatedTarget))clearFlow(node.parentElement);});
    document.addEventListener('focusin',event=>{if(event.target.matches('.node'))highlightFlow(event.target);});document.addEventListener('focusout',event=>{if(event.target.matches('.node'))clearFlow(event.target.parentElement);});
    const explainLogs=()=>document.querySelectorAll('.log-row,.log-entry').forEach(row=>{const hint=explainError(row.textContent);if(hint){row.dataset.help=hint;row.removeAttribute('title');row.tabIndex=0;row.setAttribute('aria-description',hint);}});
    document.addEventListener('pointermove',e=>{const row=e.target.closest('[data-help]');if(row)showTooltip(row,row.dataset.help,e);});
    document.addEventListener('pointerout',e=>{if(e.target.closest('[data-help]'))hideTooltip();});
    document.addEventListener('focusin',e=>{if(e.target.dataset.help)showTooltip(e.target,e.target.dataset.help);});
    document.addEventListener('focusout',hideTooltip);document.addEventListener('keydown',e=>{if(e.key==='Escape')hideTooltip();});window.addEventListener('scroll',hideTooltip,true);
    new MutationObserver(explainLogs).observe(document.querySelector('main')||document.body,{childList:true,subtree:true});explainLogs();
    new MutationObserver(decorateGraphs).observe(document.querySelector('main')||document.body,{childList:true,subtree:true});decorateGraphs();
    if(!document.title.includes('Lense')){const url=lenseURL();url.pathname='/api/topology';fetch(url,{signal:AbortSignal.timeout(4000)}).then(r=>{if(!r.ok)throw Error('Topology unavailable');return r.json();}).then(setTopology).catch(()=>{});}
    const style=document.createElement('style');style.textContent=`
      .home-link{display:block!important;color:var(--text)!important;text-decoration:none!important;padding:0!important;border:0!important;background:none!important;margin:0 0 20px!important}
      .health-compact{display:grid!important;grid-template-columns:190px minmax(0,1fr)!important;align-items:start;gap:12px;margin:10px 0!important}.health-compact>.card:first-child{display:block;padding:10px;margin:0}.health-compact .health-pie{width:64px;height:64px;margin:10px auto;border-radius:50%}.health-compact canvas{width:170px!important;height:110px!important}.health-compact>.card:first-child h3{font-size:12px}.health-compact>.card:last-child{padding:12px 16px;margin:0}.health-compact h3{font-size:13px;margin:0 0 8px;color:#a6b2c3}
      .health-compact [id$="HealthDetails"],.health-compact #healthDetails{display:flex;flex-wrap:wrap;gap:8px 24px}.health-compact .health-item{border:0;padding:3px 0;gap:12px;font-size:13px}
      .kpi{padding:12px 14px!important}.kpi .v{font-size:20px!important;margin-top:3px}.card h2{font-size:21px}.card h3{font-size:16px;margin:4px 0 14px}
      .node-actions{display:flex;flex-wrap:wrap;gap:8px}.node-actions a{display:block;color:#d9e6f5;border:1px solid #465366;border-radius:6px;padding:7px 10px;text-decoration:none;font-size:13px}.flow-muted{opacity:.16!important}.node,svg path{transition:opacity .16s}.node.presense-simulation{border-style:dashed!important}.node{cursor:pointer;box-shadow:none!important}.node:focus-visible{outline:2px solid #7dd3fc;outline-offset:3px}.node.presense{border-color:#8a9099!important}.node.sense{border-color:#14b8a6!important}.node.lense{border-color:#ff8a1d!important}.node.dispense{border-color:#ef4444!important}.node.system{border-color:#facc15!important}
      .node-dialog{background:#1b2028;color:#e6edf5;border:1px solid #465366;border-radius:12px;padding:24px;width:min(460px,85vw)}.node-dialog::backdrop{background:#0008}.node-dialog p{color:#a6b2c3;overflow-wrap:anywhere}.close-detail{float:right}.node-dialog button{cursor:pointer;background:#263445;border:1px solid #536174;color:#e6edf5;border-radius:7px;padding:8px 12px}
      .compact-list{width:100%;border-collapse:collapse;font-size:13px}.compact-list th{text-align:left;color:#9aa8b9;font-size:12px;font-weight:500}.compact-list td,.compact-list th{padding:10px 12px;border-bottom:1px solid #ffffff12}.compact-list td{overflow-wrap:anywhere}.compact-list tbody tr:hover{background:#ffffff04}.compact-list a{color:#d9e6f5!important;font-weight:600!important;font-size:13px!important;text-decoration:none}.compact-list a:hover{text-decoration:underline;color:#7dd3fc!important}.list-scroll{overflow:auto}.compact-list .actions-cell{white-space:nowrap}.compact-list button{font-size:12px;padding:6px 9px;margin:0 4px}
      .agent-group,.config-group{border-left-width:2px!important;box-shadow:none;padding:12px!important}.agent-group h3,.config-group h3{font-size:14px;color:#a6b2c3}.component-group .overview-grid{display:block!important}.component-group .overview-card{display:grid;grid-template-columns:minmax(170px,1fr) 120px minmax(180px,2fr);gap:16px;border:0;border-bottom:1px solid #ffffff12;border-radius:0;padding:9px 0;background:none;align-items:center}.component-group .overview-value{font-size:13px!important;font-weight:500;margin:0}.component-group .overview-hint{margin:0}.component-group h4{margin:12px 0 4px!important;font-size:12px}
      .config-form{margin:8px 0!important}.config-form>summary{font-size:14px}.config-actions button.local-action{border-color:#465366!important}.config-fields{gap:8px!important}
      .config-stack{grid-template-columns:1fr!important}.config-stack pre{height:auto!important;min-height:0!important;max-height:360px}.config-stack>div:has(pre:empty){display:none}.config-stack textarea{height:300px!important;min-height:180px!important}.config-form details{margin:8px 0!important}.config-stack>div>.k{display:none}
      @media(max-width:650px){.health-compact{grid-template-columns:1fr!important}.compact-list{min-width:680px}.component-group .overview-card{grid-template-columns:1fr 90px}.component-group .overview-hint{grid-column:1/-1}.layout{grid-template-columns:170px minmax(0,1fr)!important}aside{padding:12px!important}main{padding:12px!important}}
      [hidden]{display:none!important} main,.layout>main{min-width:0} .graph-scroll{overflow:auto;max-width:100%;max-height:760px;cursor:grab;overscroll-behavior:contain;scrollbar-color:#536174 #10151d;scrollbar-width:auto;border-radius:14px;touch-action:pan-x pan-y}
      .graph-scroll.panning{cursor:grabbing;user-select:none}.graph-canvas,.connection-graph{flex:none;min-width:980px;overflow:hidden!important}
      .graph-scroll .node{box-sizing:border-box;height:84px;min-width:0;display:flex;flex-direction:column;justify-content:center;padding:12px 14px}
      .graph-scroll .node .title,.graph-scroll .node .sub{overflow-wrap:anywhere}.graph-scroll .node .sub{font-size:12px}
      .graph-lane{position:absolute;top:0;bottom:0;padding:12px 16px;box-sizing:border-box;border-right:1px solid #ffffff12;background:#ffffff02;color:#a6b2c3;font-size:12px;font-weight:650;letter-spacing:.06em;text-transform:uppercase;pointer-events:none;z-index:0}
      .edge-svg{z-index:1}.node{z-index:2}.overview-value{font-size:clamp(15px,1.2vw,20px)!important;overflow-wrap:anywhere;font-variant-numeric:tabular-nums}.overview-grid{grid-template-columns:repeat(auto-fit,minmax(190px,1fr))!important}
      .connection-facts{display:grid;grid-template-columns:repeat(auto-fit,minmax(180px,1fr));gap:0 20px;margin:0 0 16px}.connection-facts>div{padding:10px 0;border-bottom:1px solid #ffffff12}.connection-facts dt{font-size:12px;color:#97a3b4}.connection-facts dd{margin:4px 0;font-weight:600;overflow-wrap:anywhere}.connection-facts time{font-size:13px}
      .component-group h4{margin:14px 0 10px;color:#a6b2c3}.component-group .overview-grid{margin-bottom:18px}
      .config-form{border:1px solid #334155;border-radius:12px;padding:16px;margin:12px 0;background:#101720}.config-form details{margin:12px 0;padding:10px;border:1px solid #334155;border-radius:10px}.config-fields{display:grid;grid-template-columns:repeat(auto-fit,minmax(210px,1fr));gap:12px}.config-fields label{display:flex;flex-direction:column;align-items:stretch;border:0;border-radius:0;background:none;padding:0;font-size:12px;color:#a6b2c3}.config-fields input,.config-fields select{background:#0c1118;color:#e6edf5;border:1px solid #41516a;border-radius:8px;padding:9px;min-width:0;width:100%;box-sizing:border-box}.config-form summary{cursor:pointer;font-weight:650}.config-form button{margin:8px 8px 0 0;padding:8px 12px}.config-form p{color:#a6b2c3;font-size:13px}.empty-state{padding:22px;border:1px dashed #41516a;border-radius:12px;color:#a6b2c3}
      @media(max-width:900px){.health-layout{grid-template-columns:1fr!important}.grid{grid-template-columns:1fr!important}}
    `;document.head.append(style);
    document.querySelectorAll('.graph-canvas,.connection-graph').forEach(graphViewport);
    refreshGraphTraffic();setInterval(refreshGraphTraffic,60000);
  }
  const api={statusDomain,graphViewport,graphLanes,setTopology,topologyModel,relatedFlows,isSimulation,renderPresenseGraph,drawHealthPie,componentRoute,explainError,healthSegmentAt,topicLogRef,topicLogs,trafficLabel,hasLiveMessages};
  if(typeof module==='object'&&module.exports)module.exports=api;
  root.IoTWorkspace=api;
  if(typeof document!=='undefined'){if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',mount);else mount();}
})(typeof window==='undefined'?globalThis:window);
