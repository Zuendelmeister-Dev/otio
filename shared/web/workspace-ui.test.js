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
