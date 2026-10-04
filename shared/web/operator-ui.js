(function(root){
  'use strict';
  const descriptions={
    dashboard:['Overview','Check the system and follow a data path. Select a device for details.'],
    agents:['Devices and services','See which devices are reachable and open their details.'],
    analytics:['Measurements','Choose a device and measurement to see how its values change.'],
    components:['System status','Check collection, forwarding and storage connections.'],
    configurations:['Settings','Choose a component. Review changes before applying them.'],
    logs:['Event log','Warnings and errors explain what needs attention.'],
    namespace:['Topic browser','Browse the names used to exchange measurements.'],
    certificates:['Connection security','Manage certificates for encrypted device connections.'],
    config:['Settings','Check changes before applying them to this device.']
  };
  function pageDescription(hash,title){return descriptions[String(hash||'dashboard').replace(/^#/,'').split('/')[0]]||[title||'Workspace','Check the current status and open details when needed.'];}
  function freshnessSummary(data){
    if(!data || data.status==='unknown')return {tone:'unknown',label:'Updates could not be checked',detail:data?.warning||'Check the connection to Lense and try again.'};
    const items=data.items||[], warnings=items.filter(item=>item.status!=='fresh');
    if(!items.length)return {tone:'unknown',label:'No measurement sources configured',detail:'Add a source in Settings to start collecting measurements.'};
    if(warnings.length)return {tone:'warning',label:warnings.length+' source'+(warnings.length===1?' needs':'s need')+' attention',detail:'Stored values may be old or unavailable. Open the affected device to check its connection.'};
    if(data.truncated)return {tone:'unknown',label:'Some sources still need checking',detail:'The first '+items.length+' sources have recent measurements. Open other devices to check them individually.'};
    return {tone:'good',label:'Measurements are up to date',detail:items.length+' source'+(items.length===1?' has':'s have')+' recent measurements.'};
  }
  function ageLabel(seconds){if(!Number.isFinite(seconds))return 'No stored measurement';if(seconds<0)return 'Timestamp is in the future';if(seconds<60)return Math.floor(seconds)+' seconds ago';if(seconds<3600)return Math.floor(seconds/60)+' minutes ago';if(seconds<86400)return Math.floor(seconds/3600)+' hours ago';return Math.floor(seconds/86400)+' days ago';}
  function install(){
    const main=document.querySelector('main');if(!main || document.querySelector('.workspace-header'))return;
    document.querySelectorAll('aside nav a').forEach(link=>{const route=(link.getAttribute('href')||'').replace(/^#/,'');if(descriptions[route])link.textContent=descriptions[route][0];});
    const style=document.createElement('style');
    style.textContent=`
      :root{--muted:#a5adba;--operator-good:#35ce88;--operator-warning:#ffd075;--operator-unknown:#b8c3d2;color-scheme:dark}
      body{line-height:1.5}button,input,select{min-height:40px}button:disabled{opacity:.5;cursor:not-allowed}
      button:focus-visible,a:focus-visible,input:focus-visible,select:focus-visible,textarea:focus-visible,summary:focus-visible{outline:2px solid #66b9ff;outline-offset:3px}
      .workspace-header{display:flex;align-items:start;justify-content:space-between;gap:24px;margin-bottom:22px}.workspace-header h1{font-size:26px;font-weight:650;margin:0;letter-spacing:-.025em}.workspace-header p{font-size:14px;max-width:720px;color:var(--muted);margin:5px 0 0}
      .card,.kpi{box-shadow:none;border-radius:12px}.node:hover{transform:none!important}.k{font-size:13px}.small{font-size:13px}nav a.active{border-left:3px solid var(--accent);font-weight:600}
      .config-field-help{display:block;margin-top:5px;color:var(--muted);font-size:12px;font-weight:400;line-height:1.5}
      .operator-status{border-left:3px solid var(--operator-unknown);padding:14px 18px;margin-bottom:16px}.operator-status[data-tone="good"]{border-left-color:var(--operator-good)}.operator-status[data-tone="warning"]{border-left-color:var(--operator-warning)}.operator-status strong{font-size:16px;font-weight:650}.operator-status p{margin:5px 0;color:var(--muted);font-size:14px}.operator-status ul{padding-left:20px;margin:12px 0 0}.operator-status li{margin:7px 0}.operator-status time{font-size:13px;color:var(--muted)}
      .operator-actions{display:flex;gap:10px;flex-wrap:wrap;margin-top:12px}.operator-actions a{border:1px solid var(--border);border-radius:8px;padding:8px 12px;text-decoration:none;color:var(--text)}.operator-actions a:hover{border-color:var(--accent)}
      .operator-retention{padding:12px 0}.operator-retention strong{display:block}.operator-retention p{color:var(--muted);margin:5px 0}.operator-retention details{font-size:13px}
      @media(max-width:760px){.workspace-header{flex-direction:column;gap:12px}.kpis{grid-template-columns:repeat(2,minmax(0,1fr))}.config-workflow-grid{grid-template-columns:1fr}.config-workflow-grid textarea,.diff-box{min-height:260px}.layout{grid-template-columns:1fr!important}aside{position:static;border-right:0;border-bottom:1px solid var(--border)}nav{display:flex;flex-wrap:wrap;gap:4px}nav>a{margin:0}.subnav{flex-basis:100%;margin:0}main{padding:16px!important}}
      @media(prefers-reduced-motion:reduce){*,*::before,*::after{transition:none!important;scroll-behavior:auto!important}}
    `;
    document.head.append(style);
    const header=document.createElement('header');header.className='workspace-header';
    const heading=document.createElement('div'),title=document.createElement('h1'),hint=document.createElement('p');heading.append(title,hint);
    document.documentElement.dataset.appearance='dark';
    header.append(heading);main.prepend(header);
    const update=()=>{const description=pageDescription(location.hash,document.title);title.textContent=description[0];hint.textContent=description[1];};
    window.addEventListener('hashchange',update);update();
  }
  const api={pageDescription,freshnessSummary,ageLabel};
  if(typeof module==='object'&&module.exports)module.exports=api;
  root.IoTOperatorUI=api;
  if(typeof document!=='undefined'){if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',install);else install();}
})(typeof globalThis!=='undefined'?globalThis:this);
