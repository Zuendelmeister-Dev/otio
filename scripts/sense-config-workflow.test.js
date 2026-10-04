'use strict';
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const html = fs.readFileSync(path.join(__dirname, '../apps/sense/static/index.html'), 'utf8');
for(const match of html.matchAll(/<script\b[^>]*>([\s\S]*?)<\/script>/gi))if(match[1].trim())new vm.Script(match[1]);
const source = html.slice(html.indexOf('async function loadConfig(force='), html.indexOf('async function tick(){'));
const poll = html.slice(html.indexOf('async function loadConfigUpdate(){'), html.indexOf('setInterval(tick,'));
function deferred() { let resolve; const promise = new Promise(r => { resolve = r; }); return {promise, resolve}; }
function fixture() {
  const elements = Object.fromEntries(['configEditor','applyButton','validation','diff','configApplyResult'].map(id => [id, {value:'',textContent:'',innerHTML:'',disabled:false,readOnly:false,style:{},setAttribute(){},addEventListener(){}}]));
  const pending = new Map();
  const requests = [];
  const context = vm.createContext({...elements, document:{getElementById:id=>elements[id],querySelectorAll:()=>[]},
    configDirty:false, configLoaded:false, lastConfigRaw:'', proposalId:'', currentConfigRevision:'', currentConfigUpdate:null,
    configLoadVersion:0,configEditVersion:0,configValidationVersion:0,configActionInFlight:false,
    safeJson:url=>pending.get(url)?.promise||Promise.resolve({items:[]}),
    configWriteFetch:(url,request)=>{requests.push({url,request});return pending.get(url)?.promise;},
    setFeedback:()=>{},showToast:()=>{},showLoading:()=>{},hideLoading:()=>{},loadStatus:async()=>{},confirm:()=>true,
    IoTStandardChart:{escapeHtml:String}, console
  });
  vm.runInContext(source+poll,context);
  return {context,elements,pending,requests,run:code=>vm.runInContext(code,context)};
}
async function loaded(f) {
  const response=deferred();f.pending.set('/api/config',response);
  const load=f.context.loadConfig(true);
  response.resolve({raw:'{"name":"current"}',revision:'current',update:{phase:'ready',revision:'current'}});
  await load;
}
function validResponse(id) { return {ok:true,json:async()=>({valid:true,proposal:{id,diff:'changes'}})}; }
async function test() {
  {
    const f=fixture();await loaded(f);
    const response=deferred();f.pending.set('/api/config/validate',response);
    const validation=f.context.validateConfig();
    f.elements.configEditor.value='{"edited":true}';f.context.invalidateSenseConfigDraft();
    response.resolve(validResponse('old-text'));await validation;
    assert.equal(f.run('proposalId'),'');
    assert.equal(f.elements.applyButton.disabled,true);
  }
  {
    const f=fixture();await loaded(f);
    const response=deferred();f.pending.set('/api/config',response);
    const loading=f.context.loadConfig(true);
    f.elements.configEditor.value='{"edited":true}';f.context.invalidateSenseConfigDraft();
    response.resolve({raw:'{"late":true}',revision:'late'});await loading;
    assert.equal(f.elements.configEditor.value,'{"edited":true}');
  }
  {
    const f=fixture();await loaded(f);
    f.pending.set('/api/config/validate',{promise:Promise.resolve(validResponse('proposal'))});
    await f.context.validateConfig();
    const response=deferred();f.pending.set('/api/config/apply',response);
    const applying=f.context.applyProposal();
    await f.context.applyProposal();
    assert.equal(f.requests.filter(x=>x.url==='/api/config/apply').length,1,'double clicks must not submit duplicate applies');
    assert.equal(f.elements.configEditor.readOnly,true,'editing must pause while committing');
    f.pending.set('/api/config',{promise:Promise.resolve({raw:'{"new":true}',revision:'new',update:{phase:'restarting',revision:'new'}})});
    response.resolve({ok:true,json:async()=>({saved:true,status:'restarting',update:{phase:'restarting',revision:'new'}})});
    await applying;
    assert.match(f.elements.configApplyResult.textContent,/Saved.*Restarting/);
    assert.equal(f.elements.applyButton.disabled,true);
    assert.equal(f.elements.configEditor.readOnly,false,'saved changes must not leave the editor locked');
    f.pending.set('/api/status',{promise:Promise.resolve({config:{update:{phase:'connecting',revision:'new'}}})});
    await f.context.loadConfigUpdate();
    assert.match(f.elements.configApplyResult.textContent,/Saved.*Waiting/);
    f.pending.set('/api/status',{promise:Promise.resolve({config:{update:{phase:'ready',revision:'new'}}})});
    await f.context.loadConfigUpdate();
    assert.match(f.elements.configApplyResult.textContent,/active.*connected/);
  }
  {
    const f=fixture();await loaded(f);
    f.pending.set('/api/config/validate',{promise:Promise.resolve(validResponse('proposal'))});
    await f.context.validateConfig();
    f.pending.set('/api/status',{promise:Promise.resolve({config:{update:{phase:'ready',revision:'other'}}})});
    await f.context.loadConfigUpdate();
    assert.equal(f.run('proposalId'),'');
    assert.equal(f.elements.applyButton.disabled,true,'external configuration changes invalidate the proposal');
  }
  console.log('Sense configuration draft, asynchronous save and connection-status tests passed.');
}
test().catch(error=>{console.error(error);process.exitCode=1;});
