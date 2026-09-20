(function(root){
  'use strict';
  const protocols={
    'modbus-tcp':{label:'Modbus TCP',port:5020,metric:{register:0,scale:0.1}},
    opcua:{label:'OPC UA HTTP demo',port:4840,metric:{nodeId:'ns=2;s=Machine.Temperature',scale:1}},
    'lab-modbus-tcp':{label:'Modbus TCP · native lab',port:1502,connection:'modbus-tcp://protocol-lab:1502?default-unit-identifier=1',metric:{address:'holding-register:1:UINT',scale:0.01}},
    'lab-modbus-rtu-tcp':{label:'Modbus RTU tunnel',port:1503,connection:'modbus-rtu:tcp://protocol-lab:1503?default-unit-identifier=1',metric:{address:'holding-register:1:UINT',scale:0.01}},
    'lab-opcua-tcp':{label:'OPC UA Binary',port:4842,connection:'opc.tcp://protocol-lab:4842',metric:{nodeId:'ns=1;s=Temperature',scale:1}},
    'lab-s7':{label:'Siemens S7',port:102,connection:'s7://192.168.1.10:102?remote-rack=0&remote-slot=1',metric:{address:'%DB1:0:REAL',scale:1}},
    'lab-ethernet-ip':{label:'EtherNet/IP CIP',port:44818,connection:'eip://192.168.1.10:44818',metric:{address:'%Temperature:REAL',scale:1}},
    'lab-bacnet-ip':{label:'BACnet/IP',port:47808,connection:'bacnet-ip://192.168.1.10:47808',metric:{address:'0,1/85',scale:1}},
    'lab-knxnet-ip':{label:'KNXnet/IP',port:3671,connection:'knxnet-ip://192.168.1.10:3671',metric:{address:'1/2/3:DPT_Value_Temp',scale:1}},
    'lab-iec-60870-5-104':{label:'IEC 60870-5-104',port:2404,connection:'iec-60870-5-104://192.168.1.10:2404',metric:{address:'1/1',scale:1}},
    'lab-mqtt':{label:'MQTT',port:1883,connection:'tcp://mqtt:1883',metric:{address:'lab/temperature',scale:1}},
    'lab-amqp':{label:'AMQP 0-9-1',port:5672,connection:'amqp://lab:lab@rabbitmq:5672/',metric:{address:'amq.topic/lab.temperature',scale:1}}
  };
  function changeProtocol(source,type){
    const p=protocols[type];if(!p)throw Error('Unknown protocol');
    const result=JSON.parse(JSON.stringify(source));result.type=type;result.port=p.port;result.readMode='poll';
    result.options=Object.assign({},result.options);
    if(p.connection){result.options.connection=p.connection;result.options.gatewayURL=result.options.gatewayURL||'http://protocol-lab:8500';result.host=new URL(p.connection.replace('modbus-rtu:tcp:', 'tcp:')).hostname;}
    else {delete result.options.connection;delete result.options.gatewayURL;result.unitId=1;}
    result.metrics=(result.metrics||[{name:'temperature',unit:'°C'}]).map(m=>{
      const next={...m};for(const key of ['register','nodeId','address','path'])delete next[key];return Object.assign(next,p.metric);
    });return result;
  }
  function mount(editor){
    if(editor.dataset.formMounted)return;editor.dataset.formMounted='true';
    const panel=document.createElement('details');panel.className='config-form';
    const summary=document.createElement('summary');summary.textContent='Form editor · protocol-specific fields';
    const hint=document.createElement('p');hint.textContent='Form edits update the JSON draft. Validate and review changes before applying. Changing a source protocol replaces its address and scale presets.';
    const refresh=document.createElement('button');refresh.type='button';refresh.textContent='Load fields from JSON';
    const body=document.createElement('div');panel.append(summary,hint,refresh,body);editor.before(panel);
    editor.closest('.card').addEventListener('click',event=>{
      const button=event.target.closest('button');
      const invalid=panel.querySelector('input:invalid');
      if(button&&invalid&&/validate|apply/i.test(button.textContent)){event.preventDefault();event.stopImmediatePropagation();invalid.reportValidity();}
    },true);
    let draft,writing=false;
    const publish=()=>{writing=true;editor.value=JSON.stringify(draft,null,2);editor.dispatchEvent(new Event('input',{bubbles:true}));writing=false;};
    function field(parent,obj,key,path){
      const value=obj[key],label=document.createElement('label');label.textContent=key;
      let input;
      if(key==='type'&&path.length===2&&path[0]==='sources'){
        input=document.createElement('select');for(const [id,p]of Object.entries(protocols)){const o=document.createElement('option');o.value=id;o.textContent=p.label;input.append(o);}input.value=value;
        input.onchange=()=>{draft.sources[path[1]]=changeProtocol(obj,input.value);publish();draw();};
      }else if(key==='generatorMode'){
        input=document.createElement('select');for(const name of ['constant','sine','random-int','sawtooth','square','triangle']){const o=document.createElement('option');o.value=name;o.textContent=name;input.append(o);}if(![...input.options].some(o=>o.value===value)){const o=document.createElement('option');o.value=value;o.textContent=value;input.append(o);}input.value=value;input.onchange=()=>{obj[key]=input.value;publish();};
      }else{
        input=document.createElement('input');input.type=typeof value==='boolean'?'checkbox':typeof value==='number'?'number':'text';
        if(input.type==='number')input.step='any';if(input.type==='checkbox')input.checked=value;else input.value=value==null?'':String(value);
        input.oninput=()=>{if(input.type==='number'&&(input.value===''||!Number.isFinite(input.valueAsNumber))){input.setCustomValidity('Enter a finite number');return;}input.setCustomValidity('');obj[key]=input.type==='checkbox'?input.checked:input.type==='number'?input.valueAsNumber:input.value;publish();};
      }
      if(key==='protocol'&&obj.generatorMode)input.disabled=true;
      input.setAttribute('aria-label',path.concat(key).join(' / '));label.append(input);parent.append(label);
    }
    function object(parent,obj,path){
      const fields=document.createElement('div');fields.className='config-fields';parent.append(fields);
      for(const key of Object.keys(obj)){
        if(path.length===0&&obj.generatorMode&&((obj.protocol==='modbus-tcp'&&['speed','current'].includes(key))||(obj.protocol==='opcua'&&['humidity','pressure','vibration'].includes(key))))continue;
        if(obj[key]!==null&&typeof obj[key]==='object'){
          const details=document.createElement('details'),title=document.createElement('summary');title.textContent=Array.isArray(obj)?(obj[key].agentId||obj[key].name||'Entry '+(Number(key)+1)):key;details.open=path.length<2;details.append(title);parent.append(details);object(details,obj[key],path.concat(key));
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
