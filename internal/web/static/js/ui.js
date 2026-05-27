// Shared UI helpers: toast, debounce, global search dropdown
window.UI = {};

UI.showToast = function(message, type='info', timeout=4000){
    if(window.showToastNative){ return showToastNative(message,type,timeout); }
    const c = document.getElementById('toastContainer');
    if(!c){ console.warn('toast container missing'); alert(message); return }
    const t = document.createElement('div');
    t.className = 'toast '+type;
    t.setAttribute('role','status');
    t.setAttribute('aria-live','polite');
    t.textContent = message;
    c.appendChild(t);
    // ensure container announced by screen readers
    c.setAttribute('aria-live','polite');
    c.setAttribute('role','region');
    setTimeout(()=>{ t.style.opacity='0'; setTimeout(()=>{ try{ c.removeChild(t) }catch(e){} },300); }, timeout);
}

UI.debounce = function(fn, ms){ let t; return function(){ clearTimeout(t); t = setTimeout(()=>fn.apply(this, arguments), ms) }}

UI.toggleHeaderNav = function(button){
    const nav = document.querySelector('.header-nav');
    if(!nav || !button) return;
    const isOpen = nav.classList.toggle('nav-open');
    button.setAttribute('aria-expanded', isOpen ? 'true' : 'false');
    button.textContent = isOpen ? '✕' : '☰';
}

// Global search dropdown
UI.initGlobalSearch = function(inputId, dropdownId){
    const input = document.getElementById(inputId);
    const dropdown = document.getElementById(dropdownId);
    if(!input || !dropdown) return;
    // ARIA: act as combobox + listbox
    input.setAttribute('role','combobox');
    input.setAttribute('aria-autocomplete','list');
    input.setAttribute('aria-expanded','false');
    input.setAttribute('aria-controls', dropdownId);
    dropdown.setAttribute('role','listbox');
    dropdown.setAttribute('aria-hidden','true');

    input.addEventListener('input', UI.debounce(function(){
        const q = input.value.trim();
        if(q.length < 2){ dropdown.innerHTML=''; dropdown.style.display='none'; return }
        fetch('/api/v1/search?q='+encodeURIComponent(q)).then(r=> r.ok? r.json(): null).then(data=>{
            if(!data){ dropdown.innerHTML=''; dropdown.style.display='none'; return }
            const items = [];
            if(data.skus) data.skus.slice(0,5).forEach(s=> items.push({label: s.sku_id+' — '+s.name, href: '/catalogue?sku='+encodeURIComponent(s.sku_id)}));
            if(data.reports) data.reports.slice(0,5).forEach(r=> items.push({label: r.title, href: '/reports/'+r.id}));
            if(data.records) data.records.slice(0,5).forEach(rec=> items.push({label: rec.template_name+' (generated)', href: '/records'}));
            if(items.length===0){ dropdown.innerHTML='<div class="search-empty" role="option">No results</div>'; dropdown.style.display='block'; dropdown.setAttribute('aria-hidden','false'); input.setAttribute('aria-expanded','true'); return }
            dropdown.innerHTML = items.map((it, idx)=>`<a class="search-item" role="option" data-index="${idx}" href="${it.href}">${it.label}</a>`).join('');
            dropdown.style.display='block'; dropdown.setAttribute('aria-hidden','false'); input.setAttribute('aria-expanded','true');
        }).catch(()=>{ dropdown.innerHTML=''; dropdown.style.display='none'; });
    }, 250));
    document.addEventListener('click', e => { if(!input.contains(e.target) && !dropdown.contains(e.target)){ dropdown.style.display='none'; dropdown.setAttribute('aria-hidden','true'); input.setAttribute('aria-expanded','false'); } });
    // keyboard navigation
    input.addEventListener('keydown', function(e){
        const items = dropdown.querySelectorAll('.search-item');
        if(items.length === 0) return;
        const active = dropdown.querySelector('.active');
        if(e.key === 'ArrowDown'){
            e.preventDefault();
            if(!active){ items[0].classList.add('active'); items[0].focus(); } else { const next = active.nextElementSibling || items[0]; active.classList.remove('active'); next.classList.add('active'); next.focus(); }
        } else if(e.key === 'ArrowUp'){
            e.preventDefault();
            if(!active){ items[items.length-1].classList.add('active'); items[items.length-1].focus(); } else { const prev = active.previousElementSibling || items[items.length-1]; active.classList.remove('active'); prev.classList.add('active'); prev.focus(); }
        } else if(e.key === 'Escape'){
            dropdown.style.display='none'; dropdown.setAttribute('aria-hidden','true'); input.setAttribute('aria-expanded','false');
        }
    });
}

// expose shorthand
window.showToast = UI.showToast;
