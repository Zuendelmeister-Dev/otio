(function(){
 'use strict';
 function mount(){
  const nav=document.querySelector('nav')||document.querySelector('aside');
  if(nav&&!document.getElementById('protocol-lab-link')){
   const link=document.createElement('a');link.id='protocol-lab-link';link.textContent='Protocol Lab ↗';
   if(document.title.includes('Lense'))link.href='/protocols';
   else{const url=new URL(location.href);url.port='8000';url.pathname='/protocols';url.search='';url.hash='';link.href=url.href}
   nav.append(link);
  }
  const style=document.createElement('style');style.textContent='body{font-family:system-ui,-apple-system,"Segoe UI",sans-serif!important}main{line-height:1.5}.card{box-shadow:0 8px 26px #0002}button,input,select,textarea{font:inherit}button{cursor:pointer;transition:filter .15s}button:hover{filter:brightness(1.12)}button:disabled{cursor:not-allowed;opacity:.55}button:focus-visible,a:focus-visible,input:focus-visible,textarea:focus-visible,select:focus-visible{outline:2px solid #7dd3fc;outline-offset:3px}#protocol-lab-link{margin-top:18px;border:1px solid #315674;color:#7dd3fc;background:#102334}@media(max-width:650px){main{padding:14px!important}table{font-size:12px}textarea{max-width:100%}}';document.head.append(style);
 }
 if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',mount);else mount();
})();
