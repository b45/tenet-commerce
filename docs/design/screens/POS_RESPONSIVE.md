# POS responsive — laptop, tablet, dan HP

Revision `pos-responsive-0.2`, 2026-09-05. Status **proposed / static visual review**, bukan prototype atau implementasi yang telah diuji pada perangkat. Melengkapi [spesifikasi POS](POS_CASH_ONLINE.md) dan [peta kontrak](../POS_CONTRACT_MAP.md); tidak mengubah API, permission, money atau aturan recovery.

## Keputusan scope

Laptop, tablet dan HP menjadi target operasional POS tunai online sejak desain. Ini menggantikan pembatasan lama “phone checkout deferred”. Dukungan HP tidak otomatis menambah offline cash, camera scanner, printer Bluetooth, aplikasi native atau payment gateway. Seluruhnya tetap satu FE dan satu state transaksi, bukan tiga aplikasi berbeda.

Target lintas perangkat juga berlaku untuk desain feature Phase 3 berikutnya; paket ini baru mencakup POS. Inventory, procurement, ledger dan dashboard tetap perlu review responsif masing-masing.

## Layout berdasarkan ruang tersedia

Lebar di bawah dalam CSS px; bukan resolusi fisik layar atau deteksi model perangkat. Breakpoint adalah usulan yang dapat berubah setelah uji konten. Jangan mengunci height aplikasi mengikuti tinggi SVG.

| Lebar tersedia | POS-01 | POS-02 dan POS-03 | Bukti desain |
|---|---|---|---|
| ≥1200 | Katalog fleksibel + cart 400px; gap 24, margin 32 | Review/receipt dan aksi berdampingan | Desktop 1440×900 v0.1, dipertahankan |
| 600–1199 | Katalog kemudian cart dalam satu document scroll; margin 24 | Ringkasan kemudian tender/aksi, satu kolom | Tiga frame 768×1024 v0.2 |
| <600 | Tampilan katalog dan cart terpisah, margin 16 | Satu kolom; baris receipt menjadi blok label/nilai | Lima frame 390×844 v0.2, termasuk cart dan unknown |

1024×768 memakai mode stacked, bukan memaksakan dua panel sempit. HP landscape 844×390 boleh memakai mode stacked tetapi tinggi pendek tetap scroll normal. Reflow 320 CSS px harus mempertahankan teks dan kontrol; jangan mengecilkan seluruh canvas SVG untuk mengklaim reflow lulus. Desain 320, 1024 dan landscape belum dirender terpisah pada revision ini.

## Sumber per viewport

| Tugas / state | Tablet portrait | HP portrait |
|---|---|---|
| Katalog | [POS-01-T](../exports/pos-01-tablet-workspace.svg) | [POS-01-M](../exports/pos-01-phone-catalog.svg) |
| Keranjang | Bagian bawah POS-01-T | [POS-01-M-CART](../exports/pos-01-phone-cart.svg) |
| Review pembayaran | [POS-02-T](../exports/pos-02-tablet-payment.svg) | [POS-02-M](../exports/pos-02-phone-payment.svg) |
| Receipt confirmed | [POS-03-T](../exports/pos-03-tablet-result.svg) | [POS-03-M](../exports/pos-03-phone-result.svg) |
| Hasil belum diketahui | Perilaku sama, satu kolom; belum frame terpisah | [POS-03-M-UNKNOWN](../exports/pos-03-phone-unknown.svg) |

Contoh tetap 2 susu + 1 roti, total Rp40.000, tender Rp50.000, kembali Rp10.000 jika server mengonfirmasi. Jumlah produk contoh yang digambar bukan limit API. Header “katalog dimuat” menunjukkan waktu snapshot, bukan jaminan koneksi masih online. Saat runtime putus koneksi, indikator koneksi ditampilkan terpisah dengan teks.

## Navigasi, cart dan pergantian orientasi

- HP: katalog → “Lihat keranjang” → “Lanjut pembayaran” → konfirmasi → hasil. Katalog tidak memiliki shortcut langsung membayar, agar jumlah tetap direview.
- “Kembali ke katalog” mempertahankan search, kategori, posisi scroll dan cart. “Kembali ke keranjang” sebelum submit mempertahankan tender. Browser Back mempunyai makna yang sama; bukan reset transaksi.
- Tablet: satu document scroll, bukan dua kotak scroll kecil. Untuk katalog panjang sediakan tautan lompat “Ke keranjang (N)” di awal region; fokus menuju heading cart. Setelah perubahan cart, jangan memaksa scroll kembali ke atas.
- HP catalog memakai ringkasan total + “Lihat keranjang” di bawah. Saat sticky aktif, sisakan ruang setinggi bar aktual ditambah safe-area perangkat; item terakhir dan keyboard focus tidak boleh tertutup. Pada ruang vertikal sempit/keyboard terbuka, ubah menjadi bagian document flow. Tidak wajib selalu menempel layar.
- Payment submit dan tombol hasil memakai document flow, tidak floating di atas keyboard. Pengguna boleh scroll; tidak semua konten harus muat dalam satu layar.
- Resize/rotate tidak mengosongkan cart/tender, membuat key baru, memicu POST atau membuka sesi baru. Pertahankan state tugas: jika sedang review cart lalu lebar berubah, fokus tetap pada region cart yang setara. Jika kontrol pemegang fokus hilang, pindahkan fokus ke heading region tersebut.
- State transaksi disimpan terpisah dari susunan panel. Jangan merender dua form checkout aktif (desktop/mobile) dengan efek submit masing-masing. Pemisahan visual tidak mengubah tenant/user ownership.
- Setelah pending/unknown, Browser Back atau kembali ke katalog tidak membatalkan server dan tidak membuka tombol bayar untuk original command. Pemulihan setelah reload/background mengikuti G-02/G-05, belum diimplementasikan.

## Touch, keyboard layar dan informasi panjang

- Kontrol POS memiliki target 48×48px atau lebih. Pada jumlah barang, tombol minus/plus/Hapus terpisah dengan jarak 8px atau lebih. Tombol Hapus menyebut nama produk pada accessible name; jumlah 1 tidak dihapus diam-diam oleh minus.
- Label dan pesan error tidak memakai placeholder saja. Ukuran teks input dasar 16px atau lebih; pertahankan zoom pengguna. Keyboard numerik adalah bantuan input, bukan validator nominal.
- Jangan autofocus search/tender pada touch sehingga keyboard terbuka tanpa diminta. Ketika masuk halaman, fokus ke heading; user memilih field. Scanner eksternal bekerja hanya pada field search yang memang fokus. Scanner kamera tidak ditawarkan pada scope ini.
- Saat tender fokus: label, nominal dan error harus bisa digulir ke area terlihat; status bar dan keyboard tidak menutupi field. Tidak memerlukan custom numeric keypad. Enter/Done pada keyboard menyelesaikan input, tidak otomatis menagih.
- Jika keyboard menutup atau focus blur, tender tidak berubah menjadi nol. Tombol “Uang pas” mengisi nominal saja. Pending mengunci edit/submit namun tidak menjebak focus.
- Nama produk panjang wrap tanpa fixed card height. SKU/nomor transaksi panjang dapat dipecah untuk tampilan tanpa mengubah nilai yang disalin. Harga/total tidak di-ellipsis; pindahkan angka ke baris berikutnya jika perlu.
- Cart 20 item memakai scroll halaman; tampilkan jumlah unit dan total yang sama dengan state cart. Receipt panjang boleh scroll; “Transaksi baru” tidak menghapus receipt confirmed yang dibutuhkan untuk recovery.
- Baris tabel desktop berubah menjadi blok berlabel pada HP, bukan tabel 6 kolom yang diperkecil. Urutan nama → jumlah/harga → subtotal tetap sama bagi pembaca layar. Hindari dua salinan informasi aktif dalam accessibility tree.
- Pada 320px, kurangi jumlah kolom, bukan ukuran tombol. Kontrol kuantitas/Hapus boleh turun satu baris; nominal total boleh berdiri sendiri. Mode stacked juga berlaku pada browser zoom.

## Unknown, cetak dan bantuan

Layar unknown HP tidak mempunyai tombol “Bayar ulang”, “Transaksi berhasil”, nomor transaksi palsu, kembalian final atau “Periksa status” berbasis endpoint rekaan. Tombol **Detail bantuan** membuka disclosure lokal berisi waktu attempt, pemilik/tenant, reference command dan status terakhir; bukan mengirim pesan eksternal atau mengakses endpoint baru. Jangan menampilkan token, SQL atau stack trace. Reference teknis hanya dibuka oleh sesi pemilik sesuai kebijakan recovery; detail UI bukan pengganti G-02/G-05.

Kesetaraan perilaku berlaku pada laptop/tablet: ubah susunan informasi, bukan makna status. “Data pemulihan dipertahankan” hanya ditampilkan runtime bila penyimpanan benar-benar berhasil.

Cetak dari HP/tablet tetap opsional dan perlu uji browser serta printer nyata. Jika tidak tersedia, tampilkan alasan dan petunjuk membuka transaksi confirmed yang sama pada perangkat cetak yang didukung, dalam tenant/permission yang benar. Jangan otomatis mengirim receipt ke layanan luar, menjanjikan Bluetooth langsung atau mengulang checkout. Layout layar 390px tidak menggantikan print stylesheet 80mm.

## Matriks penerimaan responsive

Semua acceptance di bawah masih **belum diuji interaktif**. Render SVG hanya membuktikan penataan contoh statis.

| ID | Kondisi / tindakan | Hasil yang diwajibkan |
|---|---|---|
| R-01 | 1440 laptop, 1024 landscape dan 768 portrait; kerjakan penjualan contoh | Tugas/angka sama, tidak ada horizontal page overflow |
| R-02 | 390 HP; pindah katalog/cart/payment dan Browser Back sebelum submit | Search, posisi scroll, cart, tender tidak hilang; belum ada POST |
| R-03 | 320px atau reflow ekuivalen; nama 100 karakter dan angka besar dalam bounds | Nama terbaca lengkap, nilai utuh, kontrol touch tetap layak |
| R-04 | Cart 20 item; buka keyboard pada qty/tender | Item terakhir, label dan error dapat dijangkau; sticky tidak menutup focus |
| R-05 | Rotate 390×844 ↔ 844×390 pada cart, tender, pending | Tidak reset state atau menghasilkan mutation baru |
| R-06 | Double tap / Enter / Done setelah mengisi tender | Hanya tombol konfirmasi eksplisit memulai satu command; keyboard tidak auto-pay |
| R-07 | Background app, reconnect, 401 atau reload setelah submit | Unknown/recovery benar; tidak replay di user/tenant berbeda |
| R-08 | Keyboard eksternal dan screen reader | Urutan fokus logis, label qty/aksi jelas, status penting diumumkan tanpa duplikasi |
| R-09 | Keyboard layar terbuka, toolbar browser berubah dan area notch/home indicator | Kontrol/teks tidak tertutup; scrolling dan zoom tetap tersedia |
| R-10 | Cetak dibatalkan/tidak didukung pada mobile | Transaksi tetap confirmed; fallback tidak membuat pembayaran baru |

Rencana pengujian minimum: browser laptop, Android Chrome dan iOS Safari pada perangkat nyata bila tersedia; tablet portrait/landscape dengan touch; keyboard eksternal untuk laptop/tablet. Catat OS/browser/version, viewport CSS, zoom, perangkat input dan hasil. Emulasi membantu layout tetapi tidak menggantikan keyboard, lifecycle aplikasi, scanner atau printer nyata. Tidak ada klaim semua browser/perangkat sudah didukung.

## Syarat lanjut dan biaya lingkup

Review delapan frame baru, setujui alur compact, lalu buat satu prototype responsif untuk menguji R-01..R-06 sebelum integrasi API. Prototype harus jelas berlabel data contoh dan terpisah dari aplikasi produksi; tidak boleh menambah mock endpoint produksi. Setelah itu uji perangkat, native editor import/restore bila dipakai, dan selesaikan backend gates sebelum menerima uang nyata.

Ini menambah ruang pengujian dibanding desktop/tablet saja, bukan otomatis melipatgandakan kode. Estimasi Phase 3 lama belum dikalibrasi untuk HP sebagai target operasional; ukur satu vertical slice mobile beserta revisinya sebelum menetapkan tambahan jam. Tidak ada estimasi token atau penghematan kuota yang diklaim tanpa telemetry.
