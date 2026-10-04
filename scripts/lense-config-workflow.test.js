'use strict';
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const html = fs.readFileSync(path.join(__dirname, '../apps/lense/static/index.html'), 'utf8');
const source = html.slice(html.indexOf('let lenseConfigItems='), html.indexOf('const configGroupMeta='));
function deferred() { let resolve; const promise = new Promise(r => { resolve = r; }); return {promise, resolve}; }
function fixture() {
  const elements = new Map();
  const storage = new Map();
  const pending = new Map();
  const document = {
    getElementById(id) { if (!elements.has(id)) elements.set(id, {value:'',textContent:'',innerHTML:'',disabled:false}); return elements.get(id); },
    querySelectorAll() { return []; }
  };
  const context = vm.createContext({document, console, URL, window:{},
    IoTStandardChart:{escapeHtml:String},
    localStorage:{getItem:k=>storage.get(k)||null,setItem:(k,v)=>storage.set(k,v),removeItem:k=>storage.delete(k)},
    safeJson:url=>pending.get(url)?.promise || Promise.resolve({items:[]}),
    configWriteJson:url=>pending.get(url)?.promise || Promise.resolve({valid:true,proposal:{id:'proposal'}}),
    loadRuntime:async()=>{}, configurationBreadcrumbHtml:()=>'', toolClass:()=>'', makeDiff:(a,b)=>b,
    fixtureItems:{A:{id:'A',label:'Collector A',kind:'IoT Sense'},B:{id:'B',label:'Collector B',kind:'IoT Sense'}}
  });
  vm.runInContext(source,context);
  vm.runInContext('flattenConfigItems=()=>{lenseConfigItems=fixtureItems};historyForConfig=()=>[];storedConfigForItem=()=>"{}";',context);
  return {context, elements, pending, document, run:code=>vm.runInContext(code,context)};
}
async function load(f,id) {
  const response=deferred(); f.pending.set('/api/remote-config?id='+id,response);
  const loaded=f.context.loadConfigurationDetail(id);
  await Promise.resolve(); response.resolve({raw:JSON.stringify({name:id})}); await loaded;
}
async function test() {
  {
    const f=fixture(), a=deferred(), b=deferred();
    f.pending.set('/api/remote-config?id=A',a); f.pending.set('/api/remote-config?id=B',b);
    const loadA=f.context.loadConfigurationDetail('A'); await Promise.resolve();
    const loadB=f.context.loadConfigurationDetail('B'); await Promise.resolve();
    b.resolve({raw:'{"name":"B"}'}); await loadB;
    a.resolve({raw:'{"name":"A"}'}); await loadA;
    assert.equal(f.run('currentLenseConfigItem.id'),'B');
    assert.equal(f.document.getElementById('lenseConfigEditor').value,'{"name":"B"}');
    assert.match(f.document.getElementById('configDetailTitle').textContent,/Collector B/);
  }
  {
    const f=fixture(); await load(f,'A');
    const response=deferred(); f.pending.set('/api/remote-config/validate?id=A',response);
    const validation=f.context.validateLenseConfigDraft();
    await load(f,'B'); response.resolve({valid:true,proposal:{id:'wrong-target'}}); await validation;
    assert.equal(f.run('currentLenseRemoteProposalId'),null);
    assert.equal(f.document.getElementById('btnApplyConfigDraft').disabled,true);
  }
  {
    const f=fixture(); await load(f,'A');
    const response=deferred(); f.pending.set('/api/remote-config/validate?id=A',response);
    const validation=f.context.validateLenseConfigDraft();
    f.document.getElementById('lenseConfigEditor').value='{"edited":true}'; f.context.invalidateConfigDraft();
    response.resolve({valid:true,proposal:{id:'old-text'}}); await validation;
    assert.equal(f.run('currentLenseConfigProposal'),null);
    assert.equal(f.document.getElementById('btnApplyConfigDraft').disabled,true);
  }
  {
    const f=fixture(); await load(f,'A'); await f.context.validateLenseConfigDraft();
    let request; const response=deferred();
    f.context.configWriteJson=(url,body)=>{request={url,body};return response.promise;};
    const applying=f.context.applyLenseConfigDraft();
    // A second click cannot submit the same proposal twice.
    await f.context.applyLenseConfigDraft();
    await load(f,'B'); response.resolve({status:'ok'}); await applying;
    assert.equal(request.url,'/api/remote-config/apply?id=A');
    assert.equal(f.run('currentLenseConfigOriginal'),'{"name":"B"}');
    assert.equal(f.document.getElementById('configWorkflowState').textContent,'Idle');
  }
  {
    const f=fixture(); await load(f,'A');
    const response=deferred(); f.pending.set('/api/remote-config?id=A',response);
    const resetting=f.context.resetLenseConfigEditor();
    await load(f,'B'); response.resolve({raw:'{"oldReset":true}'}); await resetting;
    assert.equal(f.document.getElementById('lenseConfigEditor').value,'{"name":"B"}');
  }
  {
    const f=fixture(); await load(f,'A');
    const response=deferred(); f.pending.set('/api/remote-config/history?id=A',response);
    const history=f.context.renderConfigHistory(); await load(f,'B');
    response.resolve({items:[{configRaw:'WRONG HISTORY'}]}); await history;
    assert(!f.document.getElementById('lenseConfigHistory').innerHTML.includes('WRONG HISTORY'));
  }
  console.log('Lense configuration switching, editing and delayed-response tests passed');
}
test().catch(error=>{console.error(error);process.exitCode=1;});
