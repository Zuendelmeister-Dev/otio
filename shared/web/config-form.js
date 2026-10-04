(function(root){
  'use strict';
  const capabilities=typeof module==='object'&&module.exports?require('./protocol-capabilities.js'):root.IoTProtocolCapabilities;
  if(!capabilities)throw Error('Load protocol-capabilities.js before the configuration form');
  const protocols=capabilities.sources;
  const labels={pollIntervalMs:'Read interval (milliseconds)',broker:'MQTT broker',inputBroker:'Incoming MQTT broker',outputBroker:'Outgoing MQTT broker',sources:'Devices and sources',metrics:'Measurements',host:'Host or IP address',port:'Port',clientId:'Connection ID',topicPrefix:'Topic prefix',agentId:'Device ID',unitId:'Modbus unit ID',readMode:'How to receive values',subscriptionIntervalMs:'Update interval (milliseconds)',gatewayURL:'Gateway address',register:'Register address',nodeId:'Node address',scale:'Value multiplier',unit:'Measurement unit',generatorMode:'Simulation pattern',timeoutSeconds:'Request timeout (seconds)',bearerTokenEnv:'Environment variable containing the access token'};
  const hints={pollIntervalMs:'1000 milliseconds means one read per second.',clientId:'Use a different connection ID for each running instance.',topicPrefix:'The first part of the names used to publish measurements.',unitId:'The device address configured on the Modbus equipment.',scale:'Multiply the raw device value by this number to get the displayed value.'};
  function fieldLabel(key,path){if(key==='type')return path[0]==='sources'&&path.length===2?'Protocol':path.includes('metrics')?'Measurement kind':path.at(-1)==='output'?'Output method':'Type';return labels[key]||key.replace(/([a-z0-9])([A-Z])/g,'$1 $2').replace(/[_-]/g,' ').replace(/^./,char=>char.toUpperCase());}
  function changeProtocol(source,type){
    const p=protocols[type];if(!p)throw Error('Unknown protocol');
    const result=JSON.parse(JSON.stringify(source));result.type=type;result.port=p.port;result.readMode=p.poll?'poll':'subscription';
    if(!p.subscription)delete result.subscriptionIntervalMs;
    result.options=Object.assign({},result.options);
    if(p.connection){result.options.connection=p.connection;result.options.gatewayURL=result.options.gatewayURL||'http://protocol-lab:8500';result.host=new URL(p.connection.replace('modbus-rtu:tcp:', 'tcp:')).hostname;}
    else {delete result.options.connection;delete result.options.gatewayURL;if(p.requiresUnitId)result.unitId=1;}
    result.metrics=(result.metrics||[{name:'temperature',unit:'°C'}]).map(m=>{
      const next={...m};for(const key of ['register','nodeId','address','path'])delete next[key];return Object.assign(next,p.metric);
    });return result;
  }
  function mount(editor){
    if(editor.dataset.formMounted)return;editor.dataset.formMounted='true';
    const panel=document.createElement('details');panel.className='config-form';panel.open=true;
    const summary=document.createElement('summary');summary.textContent='Form editor · protocol-specific fields';
    const hint=document.createElement('p');hint.textContent='Form edits update the JSON draft. Validate and review changes before applying. Changing a source protocol replaces its address and scale presets.';
    const refresh=document.createElement('button');refresh.type='button';refresh.textContent='Load fields from JSON';
    const body=document.createElement('div');panel.append(summary,hint,refresh,body);editor.before(panel);editor.closest('.two-col')?.classList.add('config-stack');
    const json=document.createElement('details');const jsonTitle=document.createElement('summary');jsonTitle.textContent='Advanced · JSON draft';json.append(jsonTitle);editor.before(json);json.append(editor);
    editor.closest('.card').addEventListener('click',event=>{
      const button=event.target.closest('button');
      const invalid=panel.querySelector('input:invalid');
      if(button&&invalid&&/validate|apply/i.test(button.textContent)){event.preventDefault();event.stopImmediatePropagation();invalid.reportValidity();}
    },true);
    let draft,writing=false;
    const publish=()=>{writing=true;editor.value=JSON.stringify(draft,null,2);editor.dispatchEvent(new Event('input',{bubbles:true}));writing=false;};
    function field(parent,obj,key,path){
      const value=obj[key],label=document.createElement('label');label.textContent=fieldLabel(key,path);
      let input;
      if(key==='type'&&path.length===2&&path[0]==='sources'){
        input=document.createElement('select');for(const [id,p]of Object.entries(protocols)){const o=document.createElement('option');o.value=id;o.textContent=p.label;input.append(o);}input.value=value;
        input.onchange=()=>{draft.sources[path[1]]=changeProtocol(obj,input.value);publish();draw();};
      }else if(key==='readMode'&&path.length===2&&path[0]==='sources'&&protocols[obj.type]){
        input=document.createElement('select');const p=protocols[obj.type];
        for(const mode of ['poll','subscription'].filter(mode=>p[mode])){const option=document.createElement('option');option.value=mode;option.textContent=mode==='poll'?'Read periodically':'Receive updates';input.append(option);}
        if(![...input.options].some(option=>option.value===value)){const option=document.createElement('option');option.value=value;option.textContent=value+' (unsupported)';input.append(option);}
        input.value=value;input.onchange=()=>{obj[key]=input.value;publish();};
      }else if(key==='generatorMode'){
        input=document.createElement('select');for(const name of ['constant','sine','random-int','sawtooth','square','triangle']){const o=document.createElement('option');o.value=name;o.textContent=name;input.append(o);}if(![...input.options].some(o=>o.value===value)){const o=document.createElement('option');o.value=value;o.textContent=value;input.append(o);}input.value=value;input.onchange=()=>{obj[key]=input.value;publish();};
      }else{
        input=document.createElement('input');input.type=typeof value==='boolean'?'checkbox':typeof value==='number'?'number':'text';
        if(input.type==='number')input.step='any';if(input.type==='checkbox')input.checked=value;else input.value=value==null?'':String(value);
        input.oninput=()=>{if(input.type==='number'&&(input.value===''||!Number.isFinite(input.valueAsNumber))){input.setCustomValidity('Enter a finite number');return;}input.setCustomValidity('');obj[key]=input.type==='checkbox'?input.checked:input.type==='number'?input.valueAsNumber:input.value;publish();};
      }
      if(key==='protocol'&&obj.generatorMode)input.disabled=true;
      input.setAttribute('aria-label',path.concat(fieldLabel(key,path)).join(' / '));label.append(input);
      if(hints[key]){const help=document.createElement('small');help.className='config-field-help';help.textContent=hints[key];label.append(help);}
      parent.append(label);
    }
    function object(parent,obj,path){
      const fields=document.createElement('div');fields.className='config-fields';parent.append(fields);
      for(const key of Object.keys(obj)){
        // Sense retains this legacy field in JSON but does not use it as a timeout.
        if(path.length===0&&key==='healthTimeoutSeconds'&&Array.isArray(obj.sources)&&obj.broker)continue;
        if(path.length===0&&obj.generatorMode&&((obj.protocol==='modbus-tcp'&&['speed','current'].includes(key))||(obj.protocol==='opcua'&&['humidity','pressure','vibration'].includes(key))))continue;
        if(obj[key]!==null&&typeof obj[key]==='object'){
          const details=document.createElement('details'),title=document.createElement('summary');title.textContent=Array.isArray(obj)?(obj[key].agentId||obj[key].name||'Entry '+(Number(key)+1)):fieldLabel(key,path);details.open=path.length===0;details.append(title);parent.append(details);object(details,obj[key],path.concat(key));
          if(Array.isArray(obj)){const remove=document.createElement('button');remove.type='button';remove.textContent='Remove entry from draft';remove.onclick=()=>{obj.splice(Number(key),1);publish();draw();};details.append(remove);}
        }else field(fields,obj,key,path);
      }
      if(Array.isArray(obj)&&(path.at(-1)==='sources'||path.at(-1)==='metrics')){
        const add=document.createElement('button');add.type='button';add.textContent=path.at(-1)==='sources'?'Add source':'Add metric';
        add.onclick=()=>{if(path.at(-1)==='sources')obj.push(changeProtocol({agentId:'source-'+(obj.length+1),host:'localhost',unitId:1,metrics:[{name:'temperature',unit:'°C',type:'gauge'}]},'modbus-tcp'));else{const source=draft.sources[Number(path[1])];obj.push({name:'metric-'+(obj.length+1),unit:'',type:'gauge',...(protocols[source.type]?.metric||{address:'',scale:1})});}publish();draw();};parent.append(add);
      }
    }
    function draw(){body.replaceChildren();object(body,draft,[]);}
    function load(){try{draft=JSON.parse(editor.value);if(!draft||typeof draft!=='object')throw Error('Configuration must be an object');draw();}catch(e){body.textContent='Cannot load form: '+e.message;}}
    refresh.onclick=load;panel.addEventListener('toggle',()=>{if(panel.open)load();});
    editor.addEventListener('input',()=>{if(!writing&&panel.open)load();});
    const descriptor=Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype,'value');
    Object.defineProperty(editor,'value',{get(){return descriptor.get.call(this);},set(value){descriptor.set.call(this,value);if(!writing&&panel.open)load();}});
  }
  root.IoTConfigForm={protocols,changeProtocol,mount};
  if(typeof module==='object'&&module.exports)module.exports={protocols,changeProtocol};
  if(typeof document!=='undefined')document.addEventListener('DOMContentLoaded',()=>{document.querySelectorAll('#configEditor,#lenseConfigEditor').forEach(mount);});
})(typeof window==='undefined'?globalThis:window);
