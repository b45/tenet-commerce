# POS tunai online — spesifikasi layar

Revision `pos-lowfi-0.1`, 2026-09-05. Status **proposed**. Scope UX POS online; screen IDs `POS-01`, `POS-02`, `POS-03`. Belum ready: G-01/G-02/G-03/G-05 pada [peta kontrak](../POS_CONTRACT_MAP.md) belum ditutup. Desain low-fi dapat direview sekarang; ini bukan persetujuan release uang nyata.

Pedoman: [FE-01 sampai FE-12, layout, forms, accessibility dan domain states](../../FRONTEND_GUIDELINES.md). [Manifest/source SVG](../README.md), [flow](../flows/pos-cash-online.mmd). Role/field/permission mengikuti peta kontrak, tidak diduplikasi sebagai kontrak baru.

## Tujuan dan contoh bersama

Kasir menjual dua susu @Rp15.000 dan satu roti @Rp10.000, menerima Rp50.000, mendapatkan receipt total Rp40.000 dan kembalian Rp10.000. Semua data contoh sintetis. Keberhasilan berarti satu transaksi server terkonfirmasi, angka receipt benar dan kasir tahu langkah berikutnya; bukan sekadar toast sukses.

## POS-01 — katalog dan keranjang

Source: [workspace](../exports/pos-01-workspace.svg). Pattern: POS workspace. Urutan region: konteks tenant/kasir → status muat → search/filter → produk → cart/total → lanjut pembayaran. Sidebar besar ditunda untuk menjaga ruang kerja kasir.

- Katalog memakai baris ringkas nama/SKU/harga/stok snapshot dan tombol “Tambah”; tidak menggunakan foto karena DTO tidak menyediakan gambar produk.
- Search berlabel “Cari nama, SKU, atau barcode”. Pencarian lokal case-insensitive nama/SKU; exact barcode mempertahankan leading zero. Scanner keyboard-wedge hanya aktif saat input pencarian fokus; Enter exact match unik menambah satu, hasil ambigu tetap ditampilkan untuk dipilih. Enter tidak pernah membayar.
- Kategori diturunkan dari snapshot; hanya kategori yang ada pada katalog dimuat, plus “Semua” dan “Tanpa kategori” bila relevan. Tidak mengarang jumlah seluruh tenant.
- Cart menggabungkan SKU, menampilkan kuantitas editable, harga/unit dan subtotal. Tambah/kurang punya nama aksesibel yang menyebut produk. Kuantitas 1 dikurangi tidak menghapus diam-diam; gunakan tombol “Hapus”. Focus pindah ke item berikutnya atau heading cart setelah penghapusan.
- Stok 0: aksi tambah disabled dengan teks “Stok habis”. Jumlah di atas snapshot ditolak secara lokal dengan penjelasan; server tetap final. Cart tidak dianggap reservasi.
- Primer “Lanjut pembayaran” hanya saat cart valid dan tersedia koneksi untuk online checkout. Snapshot timestamp adalah waktu fetch client, bukan waktu inventori server.
- Saat kembali dari pembayaran sebelum submit, cart/tender tetap ada. Refresh katalog tidak diam-diam mengganti item/harga; tampilkan perubahan dan minta review.

## POS-02 — review pembayaran tunai

Source: [payment](../exports/pos-02-payment.svg). Pattern: halaman review satu tugas, bukan modal bertumpuk. Ringkasan cart kiri dan panel tender kanan. Tidak menambah tabs QRIS/kartu/discount yang belum masuk scope.

- Masuk halaman: fetch ulang katalog, bandingkan harga/stok; perubahan terlihat dan membutuhkan review. Ini mitigasi UX, bukan solusi race G-01. Sebelum final submit, gate G-01 harus memiliki keputusan implementasi.
- Input “Uang diterima (Rp)” berlabel tetap; draft string, inputmode numeric, integer rupiah ≥ total dan dalam bounds yang disepakati. Nilai kosong berbeda dari nol. Tolak pecahan/eksponen/negatif dan format ambigu; jangan membulatkan uang input tanpa penjelasan.
- Tombol “Uang pas” mengisi total, tidak submit. Estimasi kembalian diperbarui sebagai status sopan, tidak mengumumkan setiap digit. “Konfirmasi pembayaran” mengirim hanya pada aksi eksplisit pengguna dan hanya satu command.
- Sebelum submit: persist original key/body/context. Saat pending, nonaktifkan edit dan submit ulang, tampilkan “Pembayaran sedang diproses. Jangan menagih ulang.” Tidak menyediakan “Batalkan transaksi” untuk request yang sudah dikirim.
- Navigasi/menutup browser tidak dapat dijamin terblokir. Jika pengguna keluar saat pending, simpan status dan lanjutkan layar hasil/recovery ketika kembali. Jangan menyebut menutup layar membatalkan server.
- Posisi uang fisik: kasir mengisi uang yang benar-benar diterima, memegangnya selama proses, dan menyerahkan kembalian sesuai receipt server. Pada unknown jangan menagih ulang atau otomatis mengembalikan uang; ikuti rekonsiliasi. Penanganan selisih harga setelah commit tetap dependency G-01.

## POS-03 — hasil dan receipt

Source: [result](../exports/pos-03-result.svg) menggambar state confirmed; state unknown dijelaskan di bawah, bukan disimulasikan sebagai sukses.

- Heading “Transaksi berhasil” hanya untuk response valid `COMPLETED`. Primer “Transaksi baru”; sekunder “Cetak struk”. Tampilkan nomor transaksi, waktu dengan timezone eksplisit, item snapshot receipt, subtotal/pajak/diskon/total/tender/kembalian server.
- ID kasir tersedia, nama kasir historis tidak. Header sesi boleh memakai profil saat ini tetapi jangan mengklaim nama itu snapshot historis. Merchant slug bukan legal merchant name. Frame menyebut “contoh” agar tidak dianggap bukti pembayaran nyata.
- Receipt baru dari POST dan receipt ulang dari GET memiliki bentuk DTO berbeda; adapter menormalisasi tanpa mengubah angka. Pada GET `VOIDED`, tampilkan “Dibatalkan” dan jangan tampilkan transaksi sebagai berhasil aktif.
- Cetak 80mm: stylesheet print terpisah, lebar printable dikalibrasi pada perangkat, header navigasi/tombol disembunyikan, nama panjang wrap, angka tidak terpotong. Tidak perlu instal plugin printing berbayar. Ukuran 80mm adalah target kertas, bukan jaminan printable width.
- Browser print dialog tidak membuktikan kertas tercetak. Setelah dialog tutup, jangan mengklaim “Cetak berhasil”. Sediakan “Cetak ulang” menggunakan transaksi sama tanpa POST checkout. Cetak bersifat opsional; transaksi baru boleh dimulai setelah receipt tersimpan untuk pemulihan.
- Unknown mengganti area sukses dengan “Hasil transaksi belum diketahui”, ringkasan original cart/tender berlabel belum terkonfirmasi, dan referensi bantuan lokal. Tidak menampilkan nomor transaksi palsu, kembalian final atau tombol checkout baru untuk transaksi yang sama. Record/key teknis ada di detail dukungan, bukan teks utama kasir.
- Pada baseline sebelum G-02 selesai: pesan “Jangan menagih ulang. Minta pemeriksaan transaksi.” Tidak membuat tombol status-by-key yang endpoint-nya tidak ada. Jika ID/nomor server sudah diketahui, GET detail boleh dipakai. Replay exact original command menjadi aksi otomatis hanya setelah recovery backend terbukti, dengan backoff dan ownership yang benar.

## Matriks state lintas layar

| State / trigger | Tampilan dan aksi | Data / recovery |
|---|---|---|
| Loading katalog | Skeleton struktur + “Memuat katalog”; bayar tidak tersedia | Jangan isi dengan produk palsu |
| Katalog kosong | “Belum ada produk aktif”; muat ulang | Tidak sama dengan search tanpa hasil |
| Search tidak cocok | “Produk tidak ditemukan pada katalog yang dimuat”; hapus pencarian | Cart tidak berubah |
| Cart kosong | Instruksi tambah produk; lanjut disabled beserta alasan | Search tetap dapat digunakan |
| Fetch gagal / stale | Banner dan “Muat ulang”; waktu fetch terakhir jika ada | Cart/input tidak dihapus; pembayaran perlu snapshot sesuai policy |
| Tender invalid | Error dekat field dan fokus ke field saat submit | Nilai input tetap; belum ada POST |
| Pending | Status proses, submit/edit terkunci | Command immutable tersimpan sebelum jaringan |
| Penolakan bisnis pasti | Pesan spesifik dari error code dan review cart | Jangan auto-retry konflik stok; command baru hanya setelah koreksi disetujui |
| In-flight / rate limit | “Masih diproses” / “Tunggu sebelum mencoba kembali” | Original command, backoff; bukan stok konflik |
| Outcome unknown | Bukan success/error terminal; lihat POS-03 unknown | Original command dipertahankan sampai ada bukti; G-02 |
| Session expired / forbidden | Reauth pemilik semula / akses ditolak | Kunci record; tidak replay di tenant/user lain |
| Offline sebelum submit | “Tidak terhubung. Pembayaran online belum dapat dikirim.” | Pertahankan input; tidak menawarkan menerima transaksi offline |
| Offline setelah submit | Unknown, bukan draft biasa | Jangan hapus record pembayaran saat logout |
| Confirmed | Receipt server dan kembalian | Cetak tidak membuat mutasi bisnis |
| Storage gagal sebelum submit | “Data pemulihan belum dapat disimpan. Pembayaran belum dikirim.” | Blokir submit; jangan mengaku tersimpan |

Semua state di atas adalah spesifikasi; tes interaksi belum dijalankan karena FE belum dibangun.

## Layout, komponen dan aksesibilitas

Delta responsive v0.2: [spesifikasi laptop/tablet/HP](POS_RESPONSIVE.md) dan delapan frame tambahan menjadi sumber aturan layout compact. Kontrak transaksi dokumen ini tetap berlaku.

- Frame 1440×900: margin 32, gap 24, panel kanan 400px. Background/netral grayscale; hierarki dari ukuran/posisi/weight, tidak mengunci palette brand. Wireframe bukan token system final.
- Calon reuse: ContextHeader, ConnectionStatus, SearchField, ProductRow, QuantityControl, CartSummary, AmountField, StatusPanel, Receipt. Ini kebutuhan desain, bukan komponen existing yang sudah diimplementasikan. Tidak perlu membuat semua menjadi shared sebelum pemakaian nyata.
- Baseline teks 16px, helper 14px, heading 28px, total 32px; target kontrol 48px. Pasangan utama SVG `#171717/#ffffff` dan muted `#525252/#ffffff`. Audit interactive focus/non-text/semua state masih diperlukan.
- Breakpoint usulan v0.2: ≥1200 dua panel; 600–1199 stacked satu document scroll; <600 katalog/cart terpisah. Frame desktop 1440, tablet 768 dan HP 390 tersedia; aturan 320/1024/landscape, keyboard dan orientasi ada di spec responsive. Belum diuji sebagai aplikasi interaktif.
- Search → kategori → tambah produk → jumlah/hapus → lanjut; POS-02 heading lalu field tender → uang pas → konfirmasi; POS-03 heading status → cetak → transaksi baru. Gunakan skip links untuk melewati katalog panjang. Tidak memakai shortcut huruf tunggal global.
- Focus terlihat; setelah transisi heading utama menerima focus programatik, bukan seluruh halaman diumumkan. Status pending/success pakai live region sesuai urgensi, bukan semua item cart menjadi alert. Tabel pakai HTML table semantik; bukan ARIA grid tanpa perilaku keyboard grid.
- Nama panjang wrap minimal dua baris dan tetap dapat dibaca lengkap, angka kanan dengan tabular numerals; cart 20 item scroll dengan batas jelas dan total terjangkau. Tidak mengandalkan tooltip/warna saja.
- Money locale `id-ID`, IDR utuh; timezone dari konfigurasi yang disepakati, selalu berlabel. Contoh frame memakai WIB, bukan default semua tenant yang sudah disetujui.

## Acceptance sebelum implementasi dinyatakan selesai

| ID | Given / When | Then | Evidence saat ini |
|---|---|---|---|
| A-01 | Contoh cart di atas; tender 50.000; server COMPLETED total 40.000 | Satu command, receipt 40.000, kembali 10.000 | Perhitungan contoh; belum E2E |
| A-02 | Tender kosong/39.000/pecahan; konfirmasi | Error field, input dipertahankan, nol POST | Spec saja |
| A-03 | Scan SKU sama dua kali atau barcode leading zero | Qty digabung benar; ambigu tidak auto-add | Spec saja |
| A-04 | Stok berubah sebelum checkout; 409 stok pasti | Tidak sukses; refresh dan koreksi eksplisit | Source error mapping; belum E2E |
| A-05 | Harga berubah setelah review tetapi tender masih cukup | Tidak silent overcharge; policy G-01 dipenuhi | Blocked G-01 |
| A-06 | Double click / dua tab / timeout setelah commit | Tidak double sale; original command direkonsiliasi | Blocked G-02/G-05 |
| A-07 | 401 setelah submit lalu login user lain | Command tidak direplay/dibuka dengan pemilik berbeda | Blocked G-05 |
| A-08 | Print dialog dibatalkan atau printer mati | Receipt tetap confirmed, cetak ulang tanpa checkout | Spec saja; printer belum diuji |
| A-09 | Produk panjang, cart 20 item, 200% zoom, lebar 320 | Isi/focus terbaca, angka tidak terpotong | Belum uji browser interaktif |
| A-10 | Storage penuh sebelum submit / reload setelah submit | Tidak mengirim tanpa recovery record / pulihkan status | Blocked G-05 |

## Handoff

Belum ada file aplikasi yang diimplementasikan atau API dependency yang diselesaikan. SVG perlu review pemilik proyek, lalu prototype keyboard/tablet dan pengujian usability. Tidak ada klaim WCAG compliance, native import/restore atau printer compatibility. Hasil pemeriksaan artefak lokal dicatat di [manifest](../README.md); catatan sesi/AI tetap di `docs/local/`.
