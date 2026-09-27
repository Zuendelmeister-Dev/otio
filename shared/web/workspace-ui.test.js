'use strict';
const assert=require('node:assert/strict');
const {statusDomain}=require('./workspace-ui.js');
const {changeProtocol,protocols}=require('./config-form.js');
const original={agentId:'sensor',host:'device',unitId:1,options:{custom:'keep'},metrics:[{name:'heat',register:4,scale:2,unit:'°C',custom:true}]};
const copy=JSON.stringify(original);
for(const type of Object.keys(protocols)){
 const source=changeProtocol(original,type);assert.equal(source.type,type);assert.equal(source.agentId,'sensor');assert.equal(source.options.custom,'keep');assert.equal(source.metrics[0].custom,true);assert.equal(source.metrics[0].name,'heat');assert(source.metrics[0].scale>0);
 if(type.startsWith('lab-'))assert(source.options.connection&&source.options.gatewayURL);
 if(type.includes('opcua')){assert(source.metrics[0].nodeId);assert(!('register' in source.metrics[0]));}
}
assert.equal(JSON.stringify(original),copy,'Protocol changes must not mutate the previous source');
const now=Date.parse('2026-09-16T12:00:00Z');const domain=statusDomain([{timestamp:new Date(now-10000).toISOString()}],now);assert(domain.start<=now-10000);assert(domain.end-domain.start<=31000,'Recent observations must not collapse into a 24h axis');
assert(statusDomain([{timestamp:new Date(now-3600000).toISOString()}],now).start<now-3600000);
console.log('Workspace UI and protocol form tests passed');

const {topologyModel,relatedFlows,isSimulation}=require('./workspace-ui.js');
const model=topologyModel({senses:[{id:'modbus',sourceType:'modbus-tcp'},{id:'opc',sourceType:'opcua'}],agents:[{agentId:'machine',sourceHost:'presense-modbus',senseId:'modbus'},{agentId:'external',sourceHost:'plc',senseId:'opc'}],dispenses:[{id:'out-m',sourceType:'modbus-tcp',target:'target'},{id:'out-o',sourceType:'opcua',target:'target'}],targets:[{id:'target'}]});
const find=id=>model.find(n=>n.id===id);
assert(find('machine').simulation);assert(!find('external').simulation);assert(!isSimulation({host:'real-plc'}));
assert.deepEqual(model.filter(n=>relatedFlows(find('modbus').flows,n.flows)).map(n=>n.id),['machine','modbus','out-m','target','mqtt']);
assert(!relatedFlows(find('modbus').flows,find('opc').flows),'shared broker must not merge unrelated flows');
assert(relatedFlows(find('mqtt').flows,find('opc').flows));
assert(!relatedFlows(find('modbus').flows,find('postgres').flows));
console.log('Shared topology, simulator identity and data-flow isolation tests passed');

const {componentRoute}=require('./workspace-ui.js');
assert.equal(componentRoute('iot-lense'),'dashboard');
assert.equal(componentRoute('iot-lense',true),'configurations/iot-lense');
assert.equal(componentRoute('machine/1'),'agents/machine%2F1');

const {explainError}=require('./workspace-ui.js');assert(explainError('unexpected EOF').includes('before a complete response'));assert.equal(explainError('normal message'),'');

const {healthSegmentAt,topicLogRef,topicLogs}=require('./workspace-ui.js');
assert.equal(healthSegmentAt(100,25,4),0);
assert.equal(healthSegmentAt(100,60,4),1);
assert.equal(healthSegmentAt(65,60,4),2);
assert.equal(healthSegmentAt(65,25,4),3);
assert.equal(healthSegmentAt(0,0,4),-1);
assert.equal(healthSegmentAt(85,42,0),-1);
const ref=topicLogRef('plant/line/machine/errors');
assert.deepEqual(ref,{agentId:'machine',kind:'errors'});
assert.equal(topicLogRef('plant/machine/status'),null);
assert.deepEqual(topicLogs([],ref),[]);
const logs=[{component:'machine',level:'ERROR'},{agentId:'machine',level:'INFO'},{component:'other',level:'ERROR'}];
assert.deepEqual(topicLogs(logs,ref),[logs[0]]);
assert.deepEqual(topicLogs(logs,{...ref,kind:'logs'}),logs.slice(0,2));
console.log('Health segment hit-testing and namespace error filtering passed');

const {trafficLabel,hasLiveMessages}=require('./workspace-ui.js');
assert.equal(trafficLabel({available:true,avgPerMinute:12.4}),'Ø 12.4 msg/min · 5m');
assert(trafficLabel({available:false}).includes('—'));

assert(hasLiveMessages({kind:'sense'}));assert(hasLiveMessages({id:'mqtt'}));assert(!hasLiveMessages({kind:'presense'}));
