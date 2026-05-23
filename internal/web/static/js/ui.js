// Shared UI helpers: toast, debounce, global search dropdown
window.UI = {};

UI.showToast = function(message, type='info', timeout=4000){
    if(window.showToastNative){ return showToastNative(message,type,timeout); }
    const c = document.getElementById('toastContainer');
    if(!c){ console.warn('toast container missing'); alert(message); return }
    const t = document.createElement('div');
    t.className = 'toast '+type;
    t.textContent = message;
    c.appendChild(t);
    setTimeout(()=>{ t.style.opacity='0'; setTimeout(()=>c.removeChild(t),300); }, timeout);
}

UI.debounce = function(fn, ms){ let t; return function(){ clearTimeout(t); t = setTimeout(()=>fn.apply(this, arguments), ms) }}

// Global search dropdown
UI.initGlobalSearch = function(inputId, dropdownId){
    const input = document.getElementById(inputId);
    const dropdown = document.getElementById(dropdownId);
    if(!input || !dropdown) return;
    input.addEventListener('input', UI.debounce(function(){
        const q = input.value.trim();
        if(q.length < 2){ dropdown.innerHTML=''; dropdown.style.display='none'; return }
        fetch('/api/v1/search?q='+encodeURIComponent(q)).then(r=> r.ok? r.json(): null).then(data=>{
            if(!data){ dropdown.innerHTML=''; dropdown.style.display='none'; return }
            const items = [];
            if(data.skus) data.skus.slice(0,5).forEach(s=> items.push({label: s.sku_id+' — '+s.name, href: '/catalogue?sku='+encodeURIComponent(s.sku_id)}));
            if(data.reports) data.reports.slice(0,5).forEach(r=> items.push({label: r.title, href: '/reports/'+r.id}));
            if(data.records) data.records.slice(0,5).forEach(rec=> items.push({label: rec.template_name+' (generated)', href: '/records'}));
            if(items.length===0){ dropdown.innerHTML='<div class="search-empty">No results</div>'; dropdown.style.display='block'; return }
            dropdown.innerHTML = items.map(it=>`<a class="search-item" href="${it.href}">${it.label}</a>`).join('');
            dropdown.style.display='block';
        }).catch(()=>{ dropdown.innerHTML=''; dropdown.style.display='none'; });
    }, 250));
    document.addEventListener('click', e => { if(!input.contains(e.target) && !dropdown.contains(e.target)) dropdown.style.display='none'; });
}

// expose shorthand
window.showToast = UI.showToast;
