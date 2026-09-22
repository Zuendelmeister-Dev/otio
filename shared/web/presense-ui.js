(function(){
  'use strict';
  let loaded=null, original='', proposal='';
  const editor=document.getElementById('configEditor');if(!editor)return;
  const state=document.getElementById('configState'), apply=document.getElementById('applyConfigButton');
  const panel=document.createElement('div');panel.className='config-form';
  panel.innerHTML='<h3>Simulator protocol</h3><div class="config-fields"><label>Protocol / endpoint<select id="simulatorProtocol"></select></label><label>Host reachable from Sense<input id="senseReadHost"></label><label>Configuration token (if required)<input id="presenseToken" type="password" autocomplete="off"></label></div><p id="simulatorNote"></p><button type="button" id="copySenseSource">Copy Sense source</button><pre id="senseSourcePreview"></pre>';
  editor.closest('.card').prepend(panel);
  const protocol=document.getElementById('simulatorProtocol'),host=document.getElementById('senseReadHost'),preview=document.getElementById('senseSourcePreview');
  function showSource(){if(!loaded)return;const source=JSON.parse(JSON.stringify(loaded.source));source.host=host.value.trim();preview.textContent=JSON.stringify(source,null,2);}
  host.oninput=showSource;
  document.getElementById('copySenseSource').onclick=async()=>{try{await navigator.clipboard.writeText(preview.textContent);state.textContent='Sense source copied. Add it to sources in your Sense configuration.';}catch{state.textContent='Clipboard unavailable. Select and copy the source JSON below.';}};
  protocol.onchange=()=>{
    if(protocol.value===loaded.config.protocol)return;
    const url=new URL(location.href);url.port='8000';url.pathname='/protocols';url.search='?simulate='+encodeURIComponent(protocol.value);url.hash='';
    location.href=url.href;
  };
  async function load(){
    try{
      const response=await fetch('/api/config');if(!response.ok)throw Error(await response.text());loaded=await response.json();
      original=JSON.stringify(loaded.config,null,2);editor.value=original;proposal='';apply.disabled=true;
      document.getElementById('configDiff').textContent='';state.textContent='Loaded current simulator configuration';
      protocol.replaceChildren();
      const options=[[loaded.config.protocol,loaded.config.protocol==='opcua'?'This instance · OPC UA HTTP demo':'This instance · Modbus TCP'],['modbus-tcp','Protocol Lab · Modbus TCP'],['modbus-rtu-tcp','Protocol Lab · RTU tunnel'],['opcua-tcp','Protocol Lab · OPC UA Binary'],['mqtt','Protocol Lab · MQTT'],['amqp','Protocol Lab · AMQP']];
      for(const [value,label]of options){if(value===loaded.config.protocol&&protocol.options.length)continue;const option=document.createElement('option');option.value=value;option.textContent=label;protocol.append(option);}protocol.value=loaded.config.protocol;
      if(!host.value)host.value=loaded.source.host;showSource();
      document.getElementById('simulatorNote').textContent='This instance keeps its wire protocol and listener port. Select another protocol to open its Protocol Lab simulator. '+(loaded.persistent?'Applied settings are saved on disk.':'Applied settings last until this process restarts.');
    }catch(e){state.textContent='Could not load simulator settings: '+e.message;}
  }
  window.loadConfigPage=load;
  window.resetConfigDraft=()=>{editor.value=original;proposal='';apply.disabled=true;state.textContent='Editor reset';};
  window.validateConfigDraft=()=>{
    try{const data=JSON.parse(editor.value);if(data.protocol!==loaded.config.protocol)throw Error('Choose other protocols in Protocol Lab. This listener stays '+loaded.config.protocol);for(const [key,value]of Object.entries(data)){if(typeof value==='number'&&(!Number.isFinite(value)||value<0||value>6000))throw Error(key+' must be between 0 and 6000');}proposal=JSON.stringify(data);document.getElementById('configDiff').textContent='Current:\n'+original+'\n\nProposed:\n'+JSON.stringify(data,null,2);apply.disabled=false;state.textContent='Draft ready. Review and apply; the server validates all fields.';}catch(e){proposal='';apply.disabled=true;state.textContent=e.message;}
  };
  editor.addEventListener('input',()=>{proposal='';apply.disabled=true;state.textContent='Unsaved changes — validate again';});
  apply.onclick=async()=>{
    if(!proposal)return;apply.disabled=true;
    try{const response=await fetch('/api/config',{method:'PUT',headers:{'Content-Type':'application/json','X-OTIO-Config-Token':document.getElementById('presenseToken').value},body:proposal});if(!response.ok)throw Error(await response.text());await load();state.textContent='Simulator settings applied successfully';}catch(e){state.textContent='Apply failed: '+e.message;}
  };
  document.querySelectorAll('#page-config .k').forEach(el=>{if(el.textContent.includes('write-back'))el.textContent='Settings are applied to the running generator. Use JSON or the form editor above.';});
  if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',load);else load();
})();
