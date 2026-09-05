/* Synthetic design model only. No network, credentials, storage, or production API. */
(function(root){
  'use strict';
  const MAX_AMOUNT=999999999, MAX_QTY=999;
  const baseProducts=[
    {sku:'SUSU-001',barcode:'001001',name:'Susu UHT 1 liter',category:'Minuman',price:15000,stock:24},
    {sku:'ROTI-001',barcode:'001002',name:'Roti tawar gandum',category:'Makanan',price:10000,stock:12},
    {sku:'AIR-001',barcode:'001003',name:'Air mineral 600 ml',category:'Minuman',price:5000,stock:40},
    {sku:'KOPI-001',barcode:'001004',name:'Kopi bubuk 200 gram',category:'Minuman',price:25000,stock:0}
  ];
  function createState(){return {products:baseProducts.map(p=>({...p})),cart:{},search:'',category:'Semua',tender:'',status:'draft',scenario:'success',offline:false,attempt:null,attemptCount:0,receipt:null};}
  function items(s){return s.products.filter(p=>s.cart[p.sku]).map(p=>({...p,quantity:s.cart[p.sku],subtotal:p.price*s.cart[p.sku]}));}
  function total(s){return items(s).reduce((v,p)=>v+p.subtotal,0);}
  function locked(s){return ['pending','unknown','confirmed'].includes(s.status);}
  function setQty(s,sku,qty){
    if(locked(s))return 'Transaksi ini tidak dapat diedit.';
    const p=s.products.find(p=>p.sku===sku);
    if(!p||!Number.isInteger(qty)||qty<0||qty>Math.min(MAX_QTY,p.stock))return 'Jumlah tidak valid atau melebihi stok contoh.';
    const next=total(s)-(s.cart[sku]||0)*p.price+qty*p.price;
    if(!Number.isSafeInteger(next)||next>MAX_AMOUNT)return 'Total melebihi batas nominal prototype.';
    if(qty===0)delete s.cart[sku];else s.cart[sku]=qty;
    return '';
  }
  function parseTender(v){
    if(!/^(?:\d+|[1-9]\d{0,2}(?:\.\d{3})+)$/.test(v))return null;
    const n=Number(v.replaceAll('.',''));
    return Number.isSafeInteger(n)&&n<=MAX_AMOUNT?n:null;
  }
  function start(s){
    if(locked(s))return 'Transaksi sudah dikirim dalam simulasi.';
    if(s.offline)return 'Simulasi offline: pembayaran belum dikirim.';
    if(!items(s).length)return 'Keranjang masih kosong.';
    const cash=parseTender(s.tender);
    if(cash===null)return 'Masukkan rupiah utuh, misalnya 50000 atau 50.000.';
    if(cash<total(s))return 'Uang diterima belum cukup.';
    s.attemptCount++;
    s.attempt=Object.freeze({id:'DEMO-'+s.attemptCount,items:items(s),total:total(s),cash,scenario:s.scenario});
    s.status='pending';
    return '';
  }
  function settle(s){
    if(s.status!=='pending')return;
    if(s.attempt.scenario==='pending')return;
    if(s.attempt.scenario==='unknown'){s.status='unknown';return;}
    if(s.attempt.scenario==='stock'){s.status='rejected';return;}
    s.receipt={...s.attempt,change:s.attempt.cash-s.attempt.total,number:'TXN-CONTOH-'+String(s.attemptCount).padStart(3,'0')};s.status='confirmed';
  }
  function stress(s){
    if(locked(s))return false;
    s.products=Array.from({length:20},(_,i)=>({sku:'UJI-'+String(i+1).padStart(3,'0'),barcode:'000'+i,name:i===0?'Produk contoh bernama sangat panjang untuk menguji keterbacaan kemasan keluarga dengan keterangan varian dan ukuran lengkap':'Produk uji '+(i+1),category:'Uji layout',price:i===0?123456789:10000,stock:99}));
    s.cart=Object.fromEntries(s.products.map(p=>[p.sku,1]));s.search='';s.category='Semua';s.tender='';return true;
  }
  const api={createState,items,total,locked,setQty,parseTender,start,settle,stress};
  if(typeof module!=='undefined'&&module.exports)module.exports=api;else root.TenetDemo=api;
})(globalThis);
