# POS interaction prototype v0.3

Prototype desain lokal, **bukan aplikasi kasir produksi**. Semua produk, identitas, stok, nominal dan hasil pembayaran sintetis. Jangan memasukkan data pribadi, token, kredensial atau uang nyata.

## Membuka prototype

Buka `index.html` di browser bersama tiga file pendukungnya, atau jalankan dari root repository:

```sh
python3 -m http.server 8765 --bind 127.0.0.1 --directory docs/design/prototype
```

Lalu buka `http://127.0.0.1:8765/`. Ctrl+C menghentikan server. Server hanya menyajikan folder prototype dan hanya listen di localhost; tidak membuka backend atau dokumen internal. Akses dari HP fisik memerlukan cara transfer/preview yang disepakati terpisah; jangan mengganti bind menjadi publik tanpa review.

Tidak perlu build, npm install, CDN, font unduhan, plugin, akun desain atau API AI. File HTML/CSS/JS biasa dapat dipindahkan bersama ke komputer lain. HTML interaktif tidak otomatis menjadi komponen native Penpot/Figma; SVG dan spesifikasi sebelumnya tetap sumber migrasi visual/perilaku.

## Yang dapat dicoba

1. Tambah susu dua kali dan roti sekali. Total contoh Rp40.000.
2. Pada lebar HP, buka keranjang, lanjut pembayaran; isi `50000` atau `50.000`.
3. Enter/Done pada field nominal tidak membayar. Klik “Konfirmasi pembayaran” untuk memulai simulasi satu kali. Hasil sukses memberikan kembalian Rp10.000.
4. Buka **Pengaturan uji desain** untuk memilih respons terputus, penolakan stok, tetap diproses, atau offline sebelum kirim. Kontrol ini khusus laboratorium, bukan bagian dari UI kasir produksi.
5. Coba “Muat 20 item / nama panjang”, search, barcode `001001`, Browser Back sebelum submit, perubahan lebar dan orientasi. Target review: 320, 390, 768, 1024, 1440 CSS px dan landscape 844×390.
6. Pending/unknown mengunci command walaupun hash/Browser Back berubah. Untuk keluar dari skenario yang sengaja tertahan, gunakan **Reset seluruh demo** dan konfirmasikan penghapusan data contoh.

Desktop ≥1200 memakai dua panel, 600–1199 stacked, <600 katalog/cart terpisah. Semua action bar berada di document flow agar tidak menutupi field saat keyboard muncul. Ini penyederhanaan dari kemungkinan sticky bar pada spec, bukan jaminan perilaku keyboard perangkat nyata.

## Batas yang sengaja dipertahankan

- Tidak ada request API, auth, database, idempotency backend, settlement, persetujuan harga, storage, service worker, camera scanner atau integrasi printer Bluetooth. CSP melarang koneksi keluar; source tidak memanggil fetch/XHR/WebSocket.
- Model memori hanya menguji transisi UI. Reload/reset menghapus seluruh state; “Transaksi baru” menghapus receipt demo sebelumnya. Tidak ada riwayat atau jaminan recovery. Ini **berbeda dari kewajiban produksi** mempertahankan transaksi/receipt. G-01/G-02/G-03/G-05 belum terselesaikan.
- Simulasi pending berdurasi sekitar 900ms sebelum hasil, kecuali skenario “Tetap diproses”. Tidak mengukur latency server. Respons stok sengaja tidak memutasi fixture stok; operator dapat mereview dan memilih skenario berikutnya.
- Bounds Rp999.999.999 dan qty 999 adalah batas demonstrasi, bukan kontrak bisnis yang sudah diterima.
- Cetak menggunakan dialog browser dan tetap menyertakan label prototype. Menutup dialog bukan bukti hasil printer; tidak melakukan pembayaran ulang. Tidak ada klaim mobile printing telah teruji.
- Tidak ada upgrade Next.js, pemasangan library atau perubahan produksi di `frontend/`/`backend/`. Prototype vanilla JS ini bukan perubahan arah implementasi React/TypeScript/shadcn.

## Verifikasi dan status penerimaan

```sh
node --test docs/design/prototype/model.test.cjs
node --check docs/design/prototype/model.js
node --check docs/design/prototype/prototype.js
```

Pada 2026-09-05: sembilan unit test model lulus (nominal, stok, total, single attempt, unknown, offline, rejected, pending dan stress fixture). Syntax checks lulus; server lokal memberikan HTTP 200. Hasil ini **bukan tes DOM/browser** dan tidak menutup acceptance responsive.

Browser automation yang tersedia melaporkan tidak ada browser terhubung. Karena itu klik, focus, layout/overflow, Browser Back, print, keyboard layar, screen reader dan real-device tests **belum dijalankan**. Tidak ada screenshot prototype interaktif yang diklaim telah diverifikasi. SVG v0.2 sebelumnya memiliki pemeriksaan visual tersendiri.

Gunakan [R-01..R-10](../screens/POS_RESPONSIVE.md) sebagai checklist manual. Catat viewport, browser/OS, tindakan, hasil, dan masalah. R-07 recovery produksi tetap tidak dapat disertifikasi oleh model in-memory ini. Jangan meneruskan ke integrasi pembayaran nyata hanya karena demo berjalan.
