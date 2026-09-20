(function (root) {
  'use strict';
  function statusDomain(points, now) {
    const times = points.map(p => Date.parse(p.timestamp)).filter(Number.isFinite);
    const first = times.length ? Math.min(...times) : now;
    return {start: Math.min(first - 1000, now - 30000), end: now + 1000};
  }
  function graphViewport(box) {
    if (!box) return;
    let view = box.parentElement;
    if (!view.classList.contains('graph-scroll')) {
      view = document.createElement('div'); view.className = 'graph-scroll';
      box.before(view); view.append(box);
    }
    if (view.dataset.workspacePan) return;
    view.dataset.workspacePan = 'true'; view.tabIndex = 0;
    view.setAttribute('aria-label', 'Connection graph. Scroll or drag to pan; use arrow keys when focused.');
    let drag = null, moved = false;
    view.addEventListener('pointerdown', e => {
      if (e.button !== 0 || e.pointerType === 'touch' || e.target.closest('button,input,select,a')) return;
      drag = {x:e.clientX, y:e.clientY, left:view.scrollLeft, top:view.scrollTop}; moved = false;
    });
    view.addEventListener('pointermove', e => {
      if (!drag) return;
      if (!(e.buttons & 1)) {drag=null;view.classList.remove('panning');return;}
      const dx=e.clientX-drag.x, dy=e.clientY-drag.y;
      if (Math.abs(dx)+Math.abs(dy)>5) {moved=true; view.setPointerCapture(e.pointerId); view.classList.add('panning');}
      if (moved) {view.scrollLeft=drag.left-dx; view.scrollTop=drag.top-dy; e.preventDefault();}
    });
    const stop=()=>{drag=null;view.classList.remove('panning');};
    view.addEventListener('pointerup',stop); view.addEventListener('pointercancel',stop);
    view.addEventListener('click',e=>{if(moved){e.preventDefault();e.stopImmediatePropagation();moved=false;}},true);
    view.addEventListener('keydown',e=>{if(e.target!==view)return;const shifts={ArrowLeft:-120,ArrowRight:120};if(e.key in shifts){view.scrollLeft+=shifts[e.key];e.preventDefault();}});
  }
  function graphLanes(box, lanes) {
    graphViewport(box);
    box.querySelectorAll('.graph-lane').forEach(el=>el.remove());
    lanes.forEach(({x,width,label})=>{
      const lane=document.createElement('div');lane.className='graph-lane';
      lane.style.left=x+'px';lane.style.width=width+'px';lane.textContent=label;box.prepend(lane);
    });
  }
  function mount() {
    const style=document.createElement('style');style.textContent=`
      [hidden]{display:none!important} main,.layout>main{min-width:0} .graph-scroll{overflow:auto;max-width:100%;max-height:760px;cursor:grab;overscroll-behavior:contain;scrollbar-color:#536174 #10151d;scrollbar-width:auto;border-radius:14px;touch-action:pan-x pan-y}
      .graph-scroll.panning{cursor:grabbing;user-select:none}.graph-canvas,.connection-graph{flex:none;min-width:980px;overflow:hidden!important}
      .graph-scroll .node{box-sizing:border-box;height:84px;min-width:0;display:flex;flex-direction:column;justify-content:center;padding:12px 14px}
      .graph-scroll .node .title,.graph-scroll .node .sub{overflow-wrap:anywhere}.graph-scroll .node .sub{font-size:12px}
      .graph-lane{position:absolute;top:0;bottom:0;padding:12px 16px;box-sizing:border-box;border-right:1px solid #ffffff12;background:#ffffff02;color:#a6b2c3;font-size:12px;font-weight:650;letter-spacing:.06em;text-transform:uppercase;pointer-events:none;z-index:0}
      .edge-svg{z-index:1}.node{z-index:2}.overview-value{font-size:clamp(15px,1.2vw,20px)!important;overflow-wrap:anywhere;font-variant-numeric:tabular-nums}.overview-grid{grid-template-columns:repeat(auto-fit,minmax(190px,1fr))!important}
      .connection-facts{display:grid;grid-template-columns:repeat(auto-fit,minmax(180px,1fr));gap:0 20px;margin:0 0 16px}.connection-facts>div{padding:10px 0;border-bottom:1px solid #ffffff12}.connection-facts dt{font-size:12px;color:#97a3b4}.connection-facts dd{margin:4px 0;font-weight:600;overflow-wrap:anywhere}.connection-facts time{font-size:13px}
      .component-group h4{margin:14px 0 10px;color:#a6b2c3}.component-group .overview-grid{margin-bottom:18px}
      .config-form{border:1px solid #334155;border-radius:12px;padding:16px;margin:12px 0;background:#101720}.config-form details{margin:12px 0;padding:10px;border:1px solid #334155;border-radius:10px}.config-fields{display:grid;grid-template-columns:repeat(auto-fit,minmax(210px,1fr));gap:12px}.config-fields label{display:flex;flex-direction:column;align-items:stretch;border:0;border-radius:0;background:none;padding:0;font-size:12px;color:#a6b2c3}.config-fields input,.config-fields select{background:#0c1118;color:#e6edf5;border:1px solid #41516a;border-radius:8px;padding:9px;min-width:0;width:100%;box-sizing:border-box}.config-form summary{cursor:pointer;font-weight:650}.config-form button{margin:8px 8px 0 0;padding:8px 12px}.config-form p{color:#a6b2c3;font-size:13px}.empty-state{padding:22px;border:1px dashed #41516a;border-radius:12px;color:#a6b2c3}
      @media(max-width:900px){.health-layout{grid-template-columns:1fr!important}.grid{grid-template-columns:1fr!important}}
    `;document.head.append(style);
    document.querySelectorAll('.graph-canvas,.connection-graph').forEach(graphViewport);
  }
  const api={statusDomain,graphViewport,graphLanes};
  if(typeof module==='object'&&module.exports)module.exports=api;
  root.IoTWorkspace=api;
  if(typeof document!=='undefined'){if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',mount);else mount();}
})(typeof window==='undefined'?globalThis:window);
