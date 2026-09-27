(function(){
 'use strict';
 // Pure selection logic shared by the renderer and regression tests.
 function messageView(items, search = '', selectedTopic = '') {
  const recent = (items || []).slice(0, 50);
  const query = search.trim();
  const topics = [...new Set(recent.map(item => item.topic))].sort();
  const messages = recent.filter(item =>
   String(item.topic || '').includes(query) && (!selectedTopic || item.topic === selectedTopic));
  return { recent, topics, messages, missingSelection: !!selectedTopic && !topics.includes(selectedTopic) };
 }
 if (typeof module !== 'undefined' && module.exports) module.exports = { messageView };
 if (typeof document === 'undefined') return;
 const panel=document.getElementById('page-live-messages');if(!panel)return;
 panel.innerHTML='<div class="card"><a href="#dashboard">← Dashboard</a><h2>Live Messages · <span data-label></span></h2><p data-scope class="k"></p><div class="row"><button data-refresh>Refresh</button><button data-pause>Pause</button><span class="k">Latest 50 · refresh every 5 seconds while visible</span></div><div class="row" style="margin-top:12px;gap:10px;flex-wrap:wrap"><label>Topic filter <input data-topic type="search" placeholder="Topic contains, e.g. /metrics/temperature" aria-describedby="live-topic-help" style="width:min(420px,100%);box-sizing:border-box"></label><label>Available topics <select data-topic-select style="max-width:100%;width:min(420px,100%)"><option value="">All topics</option></select></label><span data-count class="k" role="status"></span></div><p id="live-topic-help" class="k">Case-sensitive text search within the latest 50 messages.</p><p data-status role="status"></p><div data-messages></div><p class="k">In-memory snapshot, newest first. Payload previews up to 2 KiB. A restart clears the buffer; messages between refreshes may be skipped.</p></div>';
 const label=panel.querySelector('[data-label]'),scope=panel.querySelector('[data-scope]'),status=panel.querySelector('[data-status]'),list=panel.querySelector('[data-messages]'),pause=panel.querySelector('[data-pause]');
 const topic=panel.querySelector('[data-topic]'),count=panel.querySelector('[data-count]'),topicSelect=panel.querySelector('[data-topic-select]');
 let selected='',paused=false,controller=null,snapshot=null;
 function current(){try{return location.hash.startsWith('#live-messages/')?decodeURIComponent(location.hash.slice(15)):'';}catch{return '';}}
 function render(){
  if(!snapshot)return;const data=snapshot;
   list.replaceChildren();
   const query=topic.value.trim(),exact=topicSelect.value;
   const {recent:items,messages:matching}=messageView(data.items,query,exact);
   count.textContent=matching.length+' / '+items.length+' messages';
   for(const item of matching){
    const row=document.createElement('article');row.style.cssText='padding:10px 0;border-bottom:1px solid #303844';
    const title=document.createElement('div');title.textContent=new Date(item.timestamp).toLocaleTimeString()+' · '+item.topic;title.style.cssText='overflow-wrap:anywhere;font-weight:600;font-size:13px';
    const payload=document.createElement('pre');payload.style.cssText='white-space:pre-wrap;overflow-wrap:anywhere;font-size:12px;line-height:1.5;margin:8px 0 0';
    let text=item.payload;try{text=JSON.stringify(JSON.parse(text),null,2);}catch{}
    payload.textContent=text+(item.truncated?'\n[Preview truncated]':'');row.append(title,payload);list.append(row);
   }
   if(!list.childElementCount)list.textContent=items.length&&(query||exact)?'No matching topics in the latest 50 messages.':data.unavailable?.length?'No messages available from the unreachable source.':'No messages observed since process start.';
 }
 function updateTopics(){
  const chosen=topicSelect.value;
  const {topics,missingSelection}=messageView(snapshot?.items,topic.value,chosen);
  const options=[new Option('All topics','')];
  for(const name of topics)options.push(new Option(name,name));
  if(missingSelection)options.push(new Option(chosen+' (not in latest 50)',chosen));
  topicSelect.replaceChildren(...options);topicSelect.value=chosen;
 }
 topic.addEventListener('input',render);
 topicSelect.addEventListener('change',render);
 async function refresh(){
  const id=current();if(!id||document.hidden||controller)return;
  const request=new AbortController();controller=request;
  let timedOut=false;const timeout=setTimeout(()=>{timedOut=true;request.abort();},10000);
  try {
   const response=await fetch('/api/live-messages?id='+encodeURIComponent(id),{cache:'no-store',signal:request.signal});
   if(!response.ok)throw Error('Messages unavailable ('+response.status+').');
   const data=await response.json();if(current()!==id||request.signal.aborted)return;
   label.textContent=data.label||id;scope.textContent=data.scope;
   status.textContent=(data.unavailable?.length?'Unavailable: '+data.unavailable.join(', ')+' · ':'')+'Updated '+new Date(data.sampledAt).toLocaleTimeString();
   snapshot=data;updateTopics();render();
  }catch(error){if(current()===id&&(timedOut||error.name!=='AbortError'))status.textContent=(timedOut?'Request timed out.':error.message)+' Previously displayed messages may be stale.';}
  finally{clearTimeout(timeout);if(controller===request)controller=null;}
 }
 function route(){controller?.abort();controller=null;const id=current();if(id!==selected){selected=id;snapshot=null;topicSelect.value='';updateTopics();count.textContent='';list.replaceChildren();label.textContent=id;scope.textContent='';status.textContent='Loading…';paused=false;pause.textContent='Pause';}if(id)refresh();}
 panel.querySelector('[data-refresh]').onclick=refresh;
 pause.onclick=()=>{paused=!paused;pause.textContent=paused?'Resume':'Pause';if(!paused)refresh();};
 window.addEventListener('hashchange',route);
 document.addEventListener('visibilitychange',()=>{if(!document.hidden&&!paused)refresh();});
 setInterval(()=>{if(!paused)refresh();},5000);route();
})();
