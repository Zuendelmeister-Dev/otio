document.addEventListener('DOMContentLoaded',()=>{
  const panel=document.createElement('section');panel.className='card';panel.id='databaseViewer';
  const style=document.createElement('style');style.textContent=`
#databaseViewer{min-width:0;max-width:100%;box-sizing:border-box}
#dbResults{overflow:auto;max-height:520px;max-width:100%;border:1px solid #334155;border-radius:10px;scrollbar-gutter:stable;scrollbar-color:#536174 #10151d}
#dbResults:empty{display:none}#dbResults:focus-visible{outline:2px solid #f59e0b}
#dbResults table{table-layout:fixed;min-width:100%;border-collapse:separate;border-spacing:0;font-size:13px}
#dbResults th,#dbResults td{box-sizing:border-box;padding:11px 14px;text-align:left;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;border-bottom:1px solid #ffffff12;overflow-wrap:normal;vertical-align:middle;line-height:20px}
#dbResults th{position:sticky;top:0;z-index:1;background:#111923;color:#aebdd0;font-weight:600;border-bottom-color:#41516a}
#dbResults tr:nth-child(even){background:#ffffff03}#dbResults tbody tr:hover{background:#ffffff09}
#dbResults .db-number{text-align:right;font-variant-numeric:tabular-nums}#dbResults .db-null{color:#8591a3;font-style:italic}
#dbResults .db-cell{display:block;width:100%;border:0;background:transparent;color:inherit;padding:0;text-align:left;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;font:inherit;cursor:zoom-in;border-radius:0}
#dbResults .db-cell:hover{color:#7dd3fc}#dbResults .db-cell:focus-visible{outline:2px solid #7dd3fc;outline-offset:-2px}
#dbCellDialog{width:min(800px,85vw);max-height:80vh;box-sizing:border-box;background:#161e29;color:#e6edf5;border:1px solid #41516a;border-radius:14px;padding:22px}
#dbCellDialog::backdrop{background:#0009}#dbCellValue{white-space:pre-wrap;overflow-wrap:anywhere;max-height:58vh;overflow:auto;font:13px/1.6 ui-monospace,monospace}
#dbCellDialog header{display:flex;justify-content:space-between;align-items:center;gap:16px}#dbCellDialog h3{margin:0}
`;document.head.append(style);
  panel.innerHTML=`<h3>PostgreSQL · Data explorer</h3><p class="small">Read-only access to the historian database. Up to 500 rows, 2 MiB per result and 5 seconds per query.</p>
    <div class="toolbar"><label>Table <select id="dbTable" aria-label="Database table"></select></label><button id="dbRefresh">Refresh tables</button><button id="dbBrowse">Preview table</button><span id="dbName" class="k"></span></div>
    <details><summary>Columns</summary><pre id="dbColumns"></pre></details>
    <label for="dbSQL">SELECT query</label><textarea id="dbSQL" spellcheck="false" style="box-sizing:border-box;width:100%;min-height:110px;margin:8px 0;padding:12px;background:#10151d;color:#e6edf5;border:1px solid #41516a;border-radius:10px;font:14px monospace" placeholder="SELECT * FROM metric_events ORDER BY ts DESC LIMIT 200"></textarea>
    <p class="small">Supported: columns or *, one public table, WHERE comparisons joined with AND, IS NULL, LIKE / ILIKE, ORDER BY and LIMIT. No functions, joins or writes.</p>
    <div class="toolbar"><button id="dbRun">Run SELECT</button><button id="dbExport" disabled>Export displayed rows · CSV</button></div><p id="dbMessage" role="status" aria-live="polite">Select a table or enter a query.</p><p class="small">Scroll horizontally for more columns. Click a text value or JSON payload to read it in full.</p><div id="dbResults" tabindex="0" role="region" aria-label="Query results, horizontally scrollable"></div><dialog id="dbCellDialog" aria-labelledby="dbCellTitle"><header><h3 id="dbCellTitle"></h3><button id="dbCellClose" type="button">Close</button></header><pre id="dbCellValue"></pre></dialog>`;
  const el=id=>panel.querySelector('#'+id);let tables={},result=null,loaded=false,busy=false;
  async function request(path,options){const response=await fetch('/api/database/'+path,options);const data=await response.json();if(!response.ok)throw Error(data.error||'Database request failed');return data;}
  const quote=s=>'"'+s.replaceAll('"','""')+'"';
  function invalidate(){result=null;el('dbResults').replaceChildren();el('dbExport').disabled=true;}
  function selected(){const name=el('dbTable').value;el('dbColumns').textContent=(tables[name]||[]).map(c=>c.name+' · '+c.type).join('\n');el('dbSQL').value=name?'SELECT * FROM public.'+quote(name)+' LIMIT 200':'';invalidate();el('dbMessage').textContent='Query ready. Run SELECT to retrieve data.';}
  async function refresh(){try{const data=await request('tables');tables=data.tables;el('dbName').textContent=data.database;el('dbTable').replaceChildren();for(const name of Object.keys(tables).sort()){const option=document.createElement('option');option.value=option.textContent=name;el('dbTable').append(option);}loaded=true;selected();if(!Object.keys(tables).length)el('dbMessage').textContent='No public tables available.';}catch(e){el('dbMessage').textContent=e.message;}}
  el('dbCellClose').onclick=()=>el('dbCellDialog').close();
  function showValue(column,value){
    el('dbCellTitle').textContent=column;let text=value===null?'NULL':typeof value==='object'?JSON.stringify(value,null,2):String(value);
    if(typeof value==='object'||/^[\s]*[\[{]/.test(text)){try{text=JSON.stringify(JSON.parse(text),null,2);}catch{}}
    el('dbCellValue').textContent=text;el('dbCellDialog').showModal();
  }
  function renderResults(){
    const table=document.createElement('table'),group=document.createElement('colgroup');
    const widths=result.columns.map(name=>name==='id'?90:/^(ts|timestamp|.*_at)$/.test(name)?280:/payload|topic|message/.test(name)?320:200);
    table.style.width=widths.reduce((a,b)=>a+b,0)+'px';
    for(const width of widths){const col=document.createElement('col');col.style.width=width+'px';group.append(col);}table.append(group);
    const head=table.createTHead().insertRow();for(const name of result.columns){const th=document.createElement('th');th.scope='col';th.textContent=name;head.append(th);}
    const body=table.createTBody();for(const row of result.rows){const tr=body.insertRow();row.forEach((value,index)=>{
      const td=tr.insertCell(),text=value===null?'NULL':typeof value==='object'?JSON.stringify(value):String(value);
      if(value===null){td.className='db-null';td.textContent=text;}
      else if(typeof value==='number'){td.className='db-number';td.textContent=text;}
      else{const button=document.createElement('button');button.type='button';button.className='db-cell';button.textContent=text;button.title=text.length>250?text.slice(0,250)+'…':text;button.setAttribute('aria-label','Show full '+result.columns[index]+' value');button.onclick=()=>showValue(result.columns[index],value);td.append(button);}
    });}el('dbResults').append(table);
  }
  async function run(){if(busy)return;busy=true;invalidate();el('dbMessage').textContent='Reading…';for(const id of ['dbRun','dbBrowse','dbRefresh','dbTable','dbSQL'])el(id).disabled=true;
    try{result=await request('query',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({sql:el('dbSQL').value})});renderResults();el('dbMessage').textContent=result.rows.length+' rows'+(result.truncated?' · Result capped. Narrow the filter for an extract.':'');el('dbExport').disabled=false;
    }catch(e){invalidate();el('dbMessage').textContent=e.message;}finally{busy=false;for(const id of ['dbRun','dbBrowse','dbRefresh','dbTable','dbSQL'])el(id).disabled=false;}}
  el('dbRun').onclick=run;el('dbRefresh').onclick=refresh;el('dbTable').onchange=selected;el('dbBrowse').onclick=()=>{selected();run();};el('dbSQL').oninput=()=>{invalidate();el('dbMessage').textContent='Query changed. Run SELECT to refresh.';};
  el('dbExport').onclick=()=>{if(!result)return;const cell=value=>{let text=value===null?'':typeof value==='object'?JSON.stringify(value):String(value);if(typeof value==='string'&&/^[\s]*[=+\-@]/.test(text))text="'"+text;return '"'+text.replaceAll('"','""')+'"';};const csv=[result.columns,...result.rows].map(row=>row.map(cell).join(',')).join('\r\n');const url=URL.createObjectURL(new Blob(['\ufeff'+csv],{type:'text/csv;charset=utf-8'}));const a=document.createElement('a');a.href=url;a.download='historian-extract.csv';a.click();setTimeout(()=>URL.revokeObjectURL(url),1000);};
  function place(){const postgres=location.hash==='#agents/postgres';const detail=document.getElementById('page-agent-detail');for(const child of detail.children){if(child!==panel)child.hidden=postgres;}const parent=document.getElementById(postgres?'page-agent-detail':'page-components');parent.append(panel);if((postgres||location.hash==='#components')&&!loaded)refresh();}
  window.addEventListener('hashchange',place);place();
});
