/* UI-only prototype. Never copy its synthetic settlement into production. */
(()=>{
  'use strict';
  const M=window.TenetDemo, main=document.querySelector('main'), notice=document.querySelector('#notice');
  let s=M.createState(), timer, catalogScroll=0;
  const compact=matchMedia('(max-width: 599px)');
  const money=n=>new Intl.NumberFormat('id-ID',{style:'currency',currency:'IDR',maximumFractionDigits:0}).format(n);
  const esc=v=>String(v).replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
  const button=(text,action,extra='')=>`<button type="button" data-action="${action}" ${extra}>${text}</button>`;
  const say=text=>{notice.textContent=text;};
  const focus=id=>document.getElementById(id)?.focus({preventScroll:true});
  function screen(){const route=location.hash.slice(1);if(M.locked(s))return 'result';return ['cart','payment'].includes(route)?route:'catalog';}
  function go(route){if(M.locked(s))route='result';if(screen()==='catalog')catalogScroll=scrollY;if(location.hash==='#'+route)render(true);else location.hash=route;}
  function heading(title,extra=''){return `<div class="page-heading"><h1 id="page-title" tabindex="-1">${title}</h1>${extra}</div>`;}
  function lines(list){return `<ul class="receipt-lines">${list.map(p=>`<li><strong>${esc(p.name)}</strong><p class="line-total">${p.quantity} × ${money(p.price)} = ${money(p.subtotal)}</p></li>`).join('')}</ul>`;}
  function products(){
    const query=s.search.toLocaleLowerCase('id-ID').trim();
    const list=s.products.filter(p=>(s.category==='Semua'||p.category===s.category)&&(!query||[p.name,p.sku,p.barcode].some(v=>v.toLocaleLowerCase('id-ID').includes(query))));
    return list.length?list.map(p=>`<article class="product"><div class="product-info"><h3>${esc(p.name)}</h3><div class="meta muted">${esc(p.sku)} · Stok ${p.stock}</div></div><div class="price">${money(p.price)}</div>${button(p.stock?'Tambah':'Stok habis','add',`id="add-${esc(p.sku)}" data-sku="${esc(p.sku)}" aria-label="Tambah ${esc(p.name)}" ${p.stock?'':'disabled'}`)}</article>`).join(''):'<p class="panel">Produk tidak ditemukan pada katalog yang dimuat.</p>';
  }
  function cart(){
    const list=M.items(s),count=list.reduce((n,p)=>n+p.quantity,0);
    return `<section class="cart-region panel" aria-labelledby="cart-title"><h2 id="cart-title" tabindex="-1">Keranjang · ${count} unit</h2><div class="compact-only back">${button('Kembali ke katalog','catalog')}</div>${list.length?`<ul class="cart-list">${list.map(p=>`<li class="cart-item"><h3>${esc(p.name)}</h3><p class="muted">${money(p.price)} / unit</p><div class="quantity-controls">${button('−','minus',`data-sku="${p.sku}" id="minus-${p.sku}" aria-label="Kurangi ${esc(p.name)}" ${p.quantity===1?'disabled':''}`)}<input id="qty-${p.sku}" data-qty="${p.sku}" inputmode="numeric" aria-label="Jumlah ${esc(p.name)}" value="${p.quantity}">${button('+','plus',`data-sku="${p.sku}" id="plus-${p.sku}" aria-label="Tambah jumlah ${esc(p.name)}"`)}${button('Hapus','remove',`class="remove" data-sku="${p.sku}" aria-label="Hapus ${esc(p.name)}"`)}</div><p class="line-total">${money(p.subtotal)}</p></li>`).join('')}</ul>`:'<p>Tambahkan produk dari katalog.</p>'}<div class="total"><span>Estimasi total</span><span class="money">${money(M.total(s))}</span></div><p class="muted">Stok belum direservasi. Total akhir mengikuti konfirmasi.</p>${button('Lanjut pembayaran','payment',`class="primary wide" ${!list.length||s.offline?'disabled':''}`)}${s.offline?'<p class="error">Simulasi offline: pembayaran belum dapat dikirim.</p>':''}</section>`;
  }
  function workspace(view){
    const count=M.items(s).reduce((n,p)=>n+p.quantity,0);
    return heading('Penjualan baru',button('Ke keranjang ('+count+')','cart'))+`<div class="workspace" data-view="${view}"><section class="catalog-region" aria-label="Katalog produk"><div class="filters"><div><label for="search">Cari nama, SKU, atau barcode</label><input id="search" type="search" value="${esc(s.search)}" placeholder="Ketik atau scan di sini"></div><div><label for="category">Kategori</label><select id="category">${['Semua',...new Set(s.products.map(p=>p.category))].map(c=>`<option ${s.category===c?'selected':''}>${esc(c)}</option>`).join('')}</select></div></div><p class="muted">Pencarian pada katalog contoh yang dimuat.</p><div id="catalog-items" class="catalog">${products()}</div><div class="cart-nav"><p id="cart-nav-total">${count} unit · ${money(M.total(s))}</p>${button('Lihat keranjang','cart', 'class="primary"')}</div></section>${cart()}</div>`;
  }
  function payment(){return heading('Pembayaran tunai')+`<div class="back">${button('Kembali ke keranjang','cart')}</div><div class="payment-grid"><section class="panel"><h2>Ringkasan belanja</h2>${lines(M.items(s))}<div class="total"><span>Estimasi total</span><span class="money">${money(M.total(s))}</span></div><p class="muted">Data contoh · diskon Rp0 · pajak Rp0. Bukan simulasi persetujuan harga backend.</p></section><section class="panel stack" aria-label="Pembayaran tunai"><div><label for="tender">Uang diterima (Rp)</label><input class="amount" id="tender" inputmode="numeric" value="${esc(s.tender)}" aria-describedby="tender-help tender-error" autocomplete="off"><p id="tender-help" class="muted">Rupiah utuh, misalnya 50000 atau 50.000. Batas demo Rp999.999.999.</p><p id="tender-error" class="error" role="alert"></p></div>${button('Uang pas','exact')}<div><p>Estimasi kembalian</p><p id="change" class="change">${change()}</p></div>${button('Konfirmasi pembayaran','submit','class="primary"')}<p class="muted">Simulasi saja. Saat diproses, jangan menagih ulang.</p></section></div>`;}
  function change(){const n=M.parseTender(s.tender);return n!==null&&n>=M.total(s)?money(n-M.total(s)):'—';}
  function result(){
    if(s.status==='confirmed'){
      const a=s.receipt;
      return heading('Transaksi berhasil — simulasi')+`<div class="result-grid"><section class="panel"><h2>Struk contoh</h2><p>${esc(a.number)} · bukan bukti pembayaran</p>${lines(a.items)}<dl class="receipt-totals"><dt>Subtotal</dt><dd>${money(a.total)}</dd><dt>Pajak</dt><dd>Rp0</dd><dt>Diskon</dt><dd>Rp0</dd><dt>Total</dt><dd><strong>${money(a.total)}</strong></dd><dt>Uang diterima</dt><dd>${money(a.cash)}</dd></dl></section><section class="panel result-actions stack"><div><h2>Kembalian</h2><p class="change">${money(a.change)}</p></div>${button('Transaksi baru (demo)','new','class="primary"')}${button('Cetak struk contoh','print')}<p class="muted">Cetak ulang tidak membuat pembayaran baru. Menutup dialog tidak membuktikan kertas tercetak.</p></section></div>`;
    }
    const pending=s.status==='pending',a=s.attempt;
    return heading(pending?'Pembayaran sedang diproses':'Hasil transaksi belum diketahui')+`<div class="result-grid"><section class="panel stack"><div class="status-box"><h2>Jangan menagih ulang.</h2><p>${pending?'Menunggu hasil simulasi.':'Respons simulasi terputus. Minta pemeriksaan transaksi.'}</p><p>Menutup layar bukan pembatalan.</p></div>${a?lines(a.items):''}<p>Estimasi belanja: <strong>${money(a?.total||0)}</strong></p><p>Uang diterima: <strong>${money(a?.cash||0)}</strong></p><p>Belum ada nomor transaksi atau kembalian final.</p></section><section class="panel stack"><details class="help"><summary>Detail bantuan</summary><p>Reference lokal: ${esc(a?.id||'—')}</p><p>Tenant contoh / Kasir Contoh. Record hanya ada di memori demo; reload akan menghapusnya.</p></details><p>Tidak ada endpoint pemeriksaan atau retry otomatis pada prototype.</p><p class="muted">Untuk mencoba ulang desain, gunakan Reset seluruh demo pada pengaturan uji.</p></section></div>`;
  }
  function render(moveFocus=false){
    const view=screen();
    main.innerHTML=view==='payment'?payment():view==='result'?result():workspace(view);
    document.querySelector('#connection').textContent=s.offline?'Simulasi offline · katalog contoh tetap terlihat':'Katalog contoh dimuat · simulasi online';
    for(const id of ['scenario','offline','stress'])document.getElementById(id).disabled=M.locked(s);
    if(moveFocus){focus('page-title');window.scrollTo(0,view==='catalog'?catalogScroll:0);if(view==='cart'){focus('cart-title');document.getElementById('cart-title')?.scrollIntoView({block:'start'});}}
  }
  function updateCart(action,sku){
    const qty=s.cart[sku]||0,next=action==='remove'?0:action==='minus'?Math.max(1,qty-1):qty+1;
    const error=M.setQty(s,sku,next);say(error||(action==='remove'?'Produk dihapus dari keranjang.':'Keranjang diperbarui.'));
    const previous=document.activeElement.id,scroll=scrollY;
    render();window.scrollTo(0,scroll);focus(document.getElementById(previous)?previous:'cart-title');
  }
  main.addEventListener('click',event=>{
    const target=event.target.closest('[data-action]');if(!target||target.disabled)return;
    const action=target.dataset.action;
    if(['add','plus','minus','remove'].includes(action)){updateCart(action,target.dataset.sku);return;}
    if(['cart','catalog','payment'].includes(action)){say('');if(action==='payment'&&(!M.items(s).length||s.offline))return;go(action);return;}
    if(action==='exact'){s.tender=String(M.total(s));document.getElementById('tender').value=s.tender;document.getElementById('change').textContent=change();document.getElementById('tender-error').textContent='';document.getElementById('tender').removeAttribute('aria-invalid');return;}
    if(action==='submit'){
      const error=M.start(s);if(error){document.getElementById('tender-error').textContent=error;document.getElementById('tender').setAttribute('aria-invalid','true');focus('tender');return;}
      say('Pembayaran sedang diproses dalam simulasi.');render(true);location.hash='result';
      timer=setTimeout(()=>{M.settle(s);if(s.status==='rejected'){say('Simulasi penolakan stok. Tinjau keranjang; tidak ada transaksi nyata.');go('cart');}else{render(true);say(s.status==='confirmed'?'Simulasi transaksi berhasil.':s.status==='unknown'?'Hasil simulasi belum diketahui.':'Simulasi masih diproses.');}},900);return;
    }
    if(action==='new'){s.cart={};s.tender='';s.status='draft';s.attempt=null;s.receipt=null;catalogScroll=0;say('Demo transaksi baru. Prototype tidak menyimpan riwayat struk.');go('catalog');return;}
    if(action==='print'){window.print();}
  });
  main.addEventListener('input',event=>{
    if(event.target.id==='search'){s.search=event.target.value;document.getElementById('catalog-items').innerHTML=products();}
    if(event.target.id==='tender'){s.tender=event.target.value;document.getElementById('change').textContent=change();if(event.target.getAttribute('aria-invalid')){const valid=M.parseTender(s.tender)!==null&&M.parseTender(s.tender)>=M.total(s);document.getElementById('tender-error').textContent=valid?'':'Masukkan nominal rupiah yang cukup.';event.target.setAttribute('aria-invalid',String(!valid));}}
  });
  main.addEventListener('change',event=>{
    if(event.target.id==='category'){s.category=event.target.value;document.getElementById('catalog-items').innerHTML=products();}
    if(event.target.dataset.qty){const val=event.target.value,error=M.setQty(s,event.target.dataset.qty,/^\d+$/.test(val)&&Number(val)>0?Number(val):NaN);say(error||'Jumlah diperbarui.');const id=event.target.id,scroll=scrollY;render();window.scrollTo(0,scroll);focus(id);}
  });
  main.addEventListener('keydown',event=>{
    if(event.key!=='Enter')return;
    if(event.target.id==='tender'){event.preventDefault();event.target.blur();return;}
    if(event.target.id==='search'){event.preventDefault();const q=s.search.trim();const matches=s.products.filter(p=>p.sku.toLowerCase()===q.toLowerCase()||p.barcode===q);if(matches.length===1){updateCart('add',matches[0].sku);focus('search');}else say('Tidak ada kecocokan unik. Pilih produk dari hasil pencarian.');}
  });
  document.getElementById('scenario').addEventListener('change',e=>{s.scenario=e.target.value;});
  document.getElementById('offline').addEventListener('change',e=>{s.offline=e.target.checked;render();say(s.offline?'Simulasi offline aktif.':'Simulasi online aktif.');});
  document.getElementById('stress').addEventListener('click',()=>{if(M.stress(s)){catalogScroll=0;render();go('cart');say('20 produk sintetis dimuat untuk uji layout.');}});
  document.getElementById('reset').addEventListener('click',()=>{if(!window.confirm('Hapus semua data contoh dan mulai ulang demo? Tidak ada transaksi nyata.'))return;clearTimeout(timer);s=M.createState();catalogScroll=0;document.getElementById('scenario').value='success';document.getElementById('offline').checked=false;say('Demo direset.');render();go('catalog');});
  window.addEventListener('hashchange',()=>render(true));
  compact.addEventListener('change',()=>{const active=document.activeElement;if(active&&active.getClientRects().length===0){focus(screen()==='cart'?'cart-title':'page-title');}});
  render();
})();
