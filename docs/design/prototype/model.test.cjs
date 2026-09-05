const {test}=require('node:test');
const assert=require('node:assert/strict');
const M=require('./model.js');
function prepared(){const s=M.createState();M.setQty(s,'SUSU-001',2);M.setQty(s,'ROTI-001',1);s.tender='50.000';return s;}
test('contoh total/tender/kembalian konsisten',()=>{const s=prepared();assert.equal(M.total(s),40000);assert.equal(M.start(s),'');M.settle(s);assert.equal(s.status,'confirmed');assert.equal(s.receipt.change,10000);});
test('nominal kosong, pecahan, eksponen, negatif, format ambigu, overflow ditolak',()=>{for(const v of ['', '1.5','1,5','1e5','-50000','50.00',' 50000 ','1000000000'])assert.equal(M.parseTender(v),null,v);for(const v of ['50000','50.000'])assert.equal(M.parseTender(v),50000);});
test('tender kurang tidak membuat attempt',()=>{const s=prepared();s.tender='39000';assert.match(M.start(s),/cukup/);assert.equal(s.attemptCount,0);});
test('stok nol, pecahan qty dan total di luar batas ditolak',()=>{const s=prepared();assert.ok(M.setQty(s,'KOPI-001',1));assert.ok(M.setQty(s,'SUSU-001',1.5));M.stress(s);assert.ok(M.setQty(s,'UJI-001',99));});
test('double submit dan edit pending tidak mengubah original attempt',()=>{const s=prepared();M.start(s);const a=s.attempt;assert.ok(M.start(s));assert.ok(M.setQty(s,'SUSU-001',5));assert.equal(s.attemptCount,1);assert.equal(s.attempt,a);assert.equal(s.attempt.total,40000);});
test('unknown tetap terkunci tanpa confirmed receipt',()=>{const s=prepared();s.scenario='unknown';M.start(s);M.settle(s);assert.equal(s.status,'unknown');assert.equal(s.receipt,null);assert.ok(M.start(s));});
test('offline sebelum submit tidak membuat command',()=>{const s=prepared();s.offline=true;assert.ok(M.start(s));assert.equal(s.attemptCount,0);});
test('rejected dapat direview tanpa kehilangan cart/tender',()=>{const s=prepared();s.scenario='stock';M.start(s);M.settle(s);assert.equal(s.status,'rejected');assert.equal(M.total(s),40000);assert.equal(s.tender,'50.000');assert.equal(M.locked(s),false);});
test('pending fixture tetap pending dan 20 item panjang benar-benar dimuat',()=>{const s=prepared();assert.equal(M.stress(s),true);assert.equal(M.items(s).length,20);assert.ok(s.products[0].name.length>=100);s.tender=String(M.total(s));s.scenario='pending';M.start(s);M.settle(s);assert.equal(s.status,'pending');assert.equal(M.stress(s),false);});
