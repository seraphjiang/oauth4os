// oauth4os feedback widget — include on any page via <script src="/web/feedback.js"></script>
(function(){
const API=window.location.origin;
const errors=[];
const origError=console.error;
console.error=function(){errors.push({ts:new Date().toISOString(),msg:Array.from(arguments).map(String).join(' '),stack:new Error().stack});if(errors.length>50)errors.shift();origError.apply(console,arguments)};

// Styles
const css=document.createElement('style');
css.textContent=`
.fb-btn{position:fixed;bottom:20px;right:20px;width:48px;height:48px;border-radius:50%;background:#58a6ff;color:#fff;border:none;font-size:22px;cursor:grab;z-index:99998;box-shadow:0 2px 12px rgba(0,0,0,.4);display:flex;align-items:center;justify-content:center;transition:transform .15s}
.fb-btn:hover{transform:scale(1.1)}
.fb-overlay{position:fixed;inset:0;background:rgba(0,0,0,.6);z-index:99999;display:none;align-items:center;justify-content:center}
.fb-overlay.open{display:flex}
.fb-modal{background:#161b22;border:1px solid #30363d;border-radius:12px;padding:24px;max-width:480px;width:92%;max-height:90vh;overflow-y:auto;color:#e6edf3;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;font-size:13px}
.fb-modal h3{margin:0 0 12px;font-size:16px}
.fb-modal label{display:block;font-size:11px;font-weight:600;margin:10px 0 4px;color:#8b949e}
.fb-modal input,.fb-modal textarea,.fb-modal select{width:100%;padding:8px 10px;border:1px solid #30363d;border-radius:6px;background:#0d1117;color:#e6edf3;font-size:12px;font-family:inherit}
.fb-modal textarea{height:80px;resize:vertical}
.fb-types{display:flex;gap:8px;margin-bottom:8px}
.fb-type{flex:1;padding:8px;border:1px solid #30363d;border-radius:8px;text-align:center;cursor:pointer;font-size:11px;background:#0d1117;color:#8b949e}
.fb-type.sel{border-color:#58a6ff;color:#58a6ff;background:#58a6ff15}
.fb-type .icon{font-size:20px;display:block;margin-bottom:2px}
.fb-actions{display:flex;gap:8px;margin-top:14px;justify-content:flex-end}
.fb-btn-s{padding:8px 18px;border-radius:6px;font-size:12px;font-weight:600;border:1px solid #30363d;background:#161b22;color:#e6edf3;cursor:pointer}
.fb-btn-s:hover{border-color:#58a6ff}
.fb-btn-p{background:#58a6ff;color:#fff;border-color:#58a6ff}
.fb-btn-p:disabled{opacity:.5}
.fb-screenshot{max-width:100%;border-radius:6px;border:1px solid #30363d;margin-top:4px;max-height:150px;object-fit:contain}
.fb-debug{margin-top:10px;border:1px solid #30363d;border-radius:6px;overflow:hidden}
.fb-debug summary{padding:6px 10px;cursor:pointer;font-size:11px;color:#8b949e;background:#0d1117}
.fb-debug pre{padding:8px;font-size:10px;max-height:120px;overflow:auto;background:#0d1117;color:#8b949e;margin:0;white-space:pre-wrap}
.fb-toast{position:fixed;top:16px;right:16px;padding:10px 18px;border-radius:8px;font-size:12px;z-index:100000;background:#3fb950;color:#fff;animation:fbSlide .3s}
@keyframes fbSlide{from{transform:translateX(100%);opacity:0}to{transform:translateX(0);opacity:1}}
`;
document.head.appendChild(css);

// Floating button
const btn=document.createElement('button');
btn.className='fb-btn';btn.innerHTML='💬';btn.title='Send Feedback';btn.setAttribute('aria-label','Send feedback');
const saved=localStorage.getItem('fb_pos');
if(saved){const p=JSON.parse(saved);btn.style.left=p.x+'px';btn.style.top=p.y+'px';btn.style.right='auto';btn.style.bottom='auto'}
document.body.appendChild(btn);

// Drag
let dragging=false,ox,oy;
btn.addEventListener('pointerdown',e=>{dragging=true;ox=e.clientX-btn.getBoundingClientRect().left;oy=e.clientY-btn.getBoundingClientRect().top;btn.setPointerCapture(e.pointerId);btn.style.cursor='grabbing'});
btn.addEventListener('pointermove',e=>{if(!dragging)return;btn.style.left=(e.clientX-ox)+'px';btn.style.top=(e.clientY-oy)+'px';btn.style.right='auto';btn.style.bottom='auto'});
btn.addEventListener('pointerup',e=>{if(dragging){dragging=false;btn.style.cursor='grab';localStorage.setItem('fb_pos',JSON.stringify({x:parseInt(btn.style.left),y:parseInt(btn.style.top)}))}});

// Overlay
const overlay=document.createElement('div');
overlay.className='fb-overlay';
overlay.innerHTML=`<div class="fb-modal">
<h3>💬 Send Feedback</h3>
<div class="fb-types">
<div class="fb-type sel" data-t="general"><span class="icon">💡</span>General</div>
<div class="fb-type" data-t="bug"><span class="icon">🐛</span>Bug</div>
<div class="fb-type" data-t="feature"><span class="icon">✨</span>Feature</div>
</div>
<label>Title *</label><input id="fb-title" placeholder="Brief summary" required>
<label>Description</label><textarea id="fb-desc" placeholder="Details..."></textarea>
<label>Screenshot <button class="fb-btn-s" id="fb-rmshot" style="font-size:10px;padding:2px 6px;margin-left:4px;display:none">✕ Remove</button></label>
<img id="fb-shot" class="fb-screenshot" style="display:none">
<details class="fb-debug"><summary>Debug Info</summary><pre id="fb-debug"></pre></details>
<div class="fb-actions"><button class="fb-btn-s" id="fb-cancel">Cancel</button><button class="fb-btn-s fb-btn-p" id="fb-submit">Submit</button></div>
</div>`;
document.body.appendChild(overlay);

let fbType='general',screenshot=null;

overlay.querySelectorAll('.fb-type').forEach(t=>{
  t.addEventListener('click',()=>{overlay.querySelectorAll('.fb-type').forEach(x=>x.classList.remove('sel'));t.classList.add('sel');fbType=t.dataset.t})
});

function captureScreenshot(){
  try{
    const c=document.createElement('canvas');
    c.width=window.innerWidth;c.height=window.innerHeight;
    // Simple: capture visible text as placeholder since html2canvas is external dep
    const ctx=c.getContext('2d');
    ctx.fillStyle='#0d1117';ctx.fillRect(0,0,c.width,c.height);
    ctx.fillStyle='#e6edf3';ctx.font='14px sans-serif';
    ctx.fillText(document.title+' — '+window.location.pathname,20,30);
    ctx.fillStyle='#8b949e';ctx.font='11px sans-serif';
    const text=(document.body.innerText||'').substring(0,500).split('\n').slice(0,15);
    text.forEach((line,i)=>ctx.fillText(line.substring(0,100),20,50+i*16));
    screenshot=c.toDataURL('image/png');
    return screenshot;
  }catch(e){return null}
}

function collectDebug(){
  return{url:location.href,title:document.title,userAgent:navigator.userAgent,viewport:innerWidth+'x'+innerHeight,timestamp:new Date().toISOString(),errors:errors.slice(-10)};
}

btn.addEventListener('click',e=>{
  if(dragging)return;
  screenshot=captureScreenshot();
  const shot=overlay.querySelector('#fb-shot');
  const rmBtn=overlay.querySelector('#fb-rmshot');
  if(screenshot){shot.src=screenshot;shot.style.display='block';rmBtn.style.display='inline'}
  else{shot.style.display='none';rmBtn.style.display='none'}
  overlay.querySelector('#fb-debug').textContent=JSON.stringify(collectDebug(),null,2);
  overlay.querySelector('#fb-title').value='';
  overlay.querySelector('#fb-desc').value='';
  overlay.classList.add('open');
  overlay.querySelector('#fb-title').focus();
});

overlay.querySelector('#fb-rmshot').addEventListener('click',()=>{screenshot=null;overlay.querySelector('#fb-shot').style.display='none';overlay.querySelector('#fb-rmshot').style.display='none'});
overlay.querySelector('#fb-cancel').addEventListener('click',()=>overlay.classList.remove('open'));
overlay.addEventListener('click',e=>{if(e.target===overlay)overlay.classList.remove('open')});

overlay.querySelector('#fb-submit').addEventListener('click',async()=>{
  const title=overlay.querySelector('#fb-title').value.trim();
  if(!title){overlay.querySelector('#fb-title').focus();return}
  const sub=overlay.querySelector('#fb-submit');
  sub.disabled=true;sub.textContent='Sending...';
  const body={type:fbType,title,description:overlay.querySelector('#fb-desc').value,screenshot,debug:collectDebug()};
  try{
    await fetch(API+'/feedback',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});
    overlay.classList.remove('open');
    const toast=document.createElement('div');toast.className='fb-toast';toast.textContent='✅ Feedback sent — thank you!';
    document.body.appendChild(toast);setTimeout(()=>toast.remove(),3000);
  }catch(e){
    const toast=document.createElement('div');toast.className='fb-toast';toast.style.background='#f85149';toast.textContent='❌ Failed: '+e.message;
    document.body.appendChild(toast);setTimeout(()=>toast.remove(),3000);
  }
  sub.disabled=false;sub.textContent='Submit';
});

document.addEventListener('keydown',e=>{if(e.key==='Escape')overlay.classList.remove('open')});
})();
