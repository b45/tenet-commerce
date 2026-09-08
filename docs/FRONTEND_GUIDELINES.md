# Pedoman Desain dan Pengerjaan Frontend Tenet Commerce

Tanggal: 2026-09-05. Berlaku untuk rancangan dan perubahan FE berikutnya.
Pedoman ditulis dalam bahasa Indonesia agar dapat dipakai sambil belajar; nama teknis tetap disertakan untuk pencarian dokumentasi.

Dokumen ini menetapkan kaidah kerja. Palette, ukuran dan contoh layout merupakan baseline awal untuk divalidasi pada prototype, bukan klaim bahwa design system atau UI sudah diimplementasikan. Scope dan dependency delivery tetap mengikuti [rencana Phase 3](FRONTEND_PHASE3_DESIGN.md).

## 1. Cara menggunakan pedoman

**Wajib** berarti kriteria review proyek. **Default** berarti titik awal yang boleh disesuaikan dengan alasan dan bukti. **Contoh** membantu pemahaman dan tidak otomatis menjadi kontrak produk. Aturan bisnis/backend dan governance repository tetap berlaku.

Untuk satu feature: baca prinsip inti, pilih pola layar, tentukan semua state penting, gunakan komponen/token yang sudah ada, lalu uji tugas pengguna. Tidak perlu menguasai seluruh teori sebelum mulai; jangan menunda penanganan error, keyboard atau aksesibilitas sampai layar selesai dipoles.

Kualitas enterprise terlihat saat banyak role, data dan kondisi gagal tetap dapat ditangani dengan jelas. Banyak kartu, animasi atau library bukan ukuran kualitasnya.

## 2. Istilah yang perlu dipahami

| Istilah | Makna | Contoh Tenet |
|---|---|---|
| UX | Cara pengguna menyelesaikan tugas dan memahami hasilnya | Kasir tahu apakah pembayaran tersimpan, dikonfirmasi, atau perlu diperiksa |
| UI | Kontrol dan tampilan yang dipakai dalam tugas itu | Input barcode, keranjang, tombol bayar, label status |
| Information architecture | Pengelompokan informasi dan navigasi | Inventory terpisah dari procurement; menu mengikuti permission |
| User flow | Urutan tindakan dan keputusan | Scan → ubah jumlah → bayar → receipt; termasuk kegagalan |
| Wireframe | Sketsa struktur tanpa detail visual final | Posisi pencarian, produk, keranjang, status koneksi |
| Mockup | Contoh visual yang lebih detail | Warna, type scale, spacing dan isi realistis pada satu layar |
| Prototype | Rancangan yang dapat dicoba alurnya | Pindah layar dan dialog; belum berarti API/settlement bekerja |
| Design system | Aturan + tokens + komponen + pola + dokumentasi + pemeliharaan | Satu bahasa visual dan interaksi untuk seluruh domain |
| Design token | Nama stabil bagi keputusan visual | `color.text.primary`, `space.fieldGap` |
| Component | Unit UI yang dapat digunakan ulang | Button, AmountField, CertificateStatus |
| Pattern/template | Susunan komponen untuk tugas berulang | Halaman list/filter/detail atau form review/submit |
| UI kit | Koleksi aset/komponen awal | Kit Penpot membantu menggambar; belum mencakup seluruh aturan produk |
| Component library | Implementasi komponen dalam kode | Primitives, forms dan tabel di React |
| Variant/state | Variant adalah jenis; state adalah kondisi | Button destructive/primary; idle/pending/disabled |

Template dashboard yang menarik tetap harus ditinjau lisensi, semantic HTML, state, data, responsive behavior dan biaya pemeliharaannya. Screenshot saja tidak cukup untuk menjadi design system.

## 3. Material Design, Carbon dan pilihan Tenet

Material Design adalah sistem desain Google. Tema Material mencakup skema warna, tipografi dan bentuk, dengan peran seperti primary dan on-primary. Artinya, teks di atas background tombol dipilih sebagai pasangan, bukan warna yang kebetulan terlihat bagus. Panduan Android ini menjelaskan konsep tersebut; contoh Compose-nya bukan kode untuk frontend web Tenet. [Material theming](https://developer.android.com/develop/ui/compose/designsystems/material3)

Carbon menyediakan pola tabel data untuk aplikasi operasional: judul, toolbar, header, baris dan pagination. Pola ini relevan untuk inventory, PO dan ledger. Ambil alasan penataan informasinya, kemudian sesuaikan kontrak/state dengan Tenet. [Carbon data tables](https://carbondesignsystem.com/components/data-table/usage/)

Default proyek adalah satu sistem kecil milik Tenet: semantic tokens, primitives yang konsisten, dan pola halaman yang terdokumentasi. Ikuti arah shadcn/ui + Tailwind yang sudah ada di [Contributing](CONTRIBUTING.md); evaluasi dependency saat implementasi. shadcn/ui menyediakan kode komponen yang dapat dipelihara sendiri, sehingga tanggung jawab upgrade dan aksesibilitas tetap ada pada proyek. [shadcn/ui](https://ui.shadcn.com/docs)

Material Design tidak sama dengan library React bernama MUI; memilih prinsip Material tidak mengharuskan memasang MUI. Jangan memasang Material/MUI, Carbon dan shadcn sekaligus untuk tombol/form yang sama. Penambahan sistem kedua membutuhkan alasan kebutuhan yang belum terpenuhi dan rencana menghindari duplikasi.

## 4. Prinsip inti dengan contoh

| ID | Kaidah | Penerapan |
|---|---|---|
| FE-01 | Mulai dari tugas dan sumber data | Sebelum menggambar chart, pastikan metric, periode dan endpoint benar-benar tersedia |
| FE-02 | Hierarki visual mengikuti kepentingan | Total bayar dan tindakan utama lebih menonjol daripada metadata transaksi |
| FE-03 | Kelompokkan informasi terkait | Label, input dan error berdekatan; jarak antarkelompok lebih besar daripada jarak di dalamnya |
| FE-04 | Perilaku konsisten lintas domain | Posisi pencarian, istilah status, format tanggal dan pola konfirmasi tidak berubah tanpa alasan |
| FE-05 | Utamakan pengenalan daripada hafalan | Tampilkan nama produk/SKU, saldo PO dan ringkasan dampak; jangan mengharuskan pengguna mengingat ID |
| FE-06 | Tampilkan status yang benar | `Menunggu sinkronisasi` tidak sama dengan `Transaksi berhasil` |
| FE-07 | Cegah kesalahan, sediakan pemulihan | Cek jumlah/tender, jelaskan konflik stok, pertahankan data input dan key retry |
| FE-08 | Aksesibilitas sejak awal | Label, keyboard, focus, kontras dan ukuran target termasuk acceptance feature |
| FE-09 | Tampilkan detail sesuai kebutuhan | Ringkasan PO dahulu, detail dokumen saat dibuka; informasi total/risiko tetap terlihat |
| FE-10 | Pakai kembali keputusan yang diterima | Token dan komponen bersama menjadi sumber implementasi; jangan menyalin CSS berbeda setiap halaman |
| FE-11 | Sesuaikan kepadatan dengan tugas | Kasir perlu target sentuh besar; finance perlu kolom terstruktur dan angka terbaca |
| FE-12 | Ukur hasil, bukan hanya selera | Uji apakah pengguna bisa checkout dan memahami error; catat waktu, salah klik dan kebingungan |

Prinsip proximity, alignment dan similarity membantu pengguna melihat kelompok informasi. Itu alasan praktis memakai jarak dan bentuk konsisten; bukan kewajiban membuat semua layar simetris atau memakai rasio estetika tertentu.

## 5. Pemilihan dan arti warna

Makna warna bergantung konteks, budaya dan kebiasaan aplikasi. `Biru pasti membuat percaya` atau `hijau selalu berarti halal` bukan aturan universal. Tenet menetapkan makna secara konsisten dengan teks, icon dan perilaku; warna GitHub issue labels juga tidak otomatis menjadi palette UI.

| Peran | Default arah warna | Pemakaian dan batas |
|---|---|---|
| Surface/background | Putih dan slate sangat terang | Area kerja; warna netral membantu informasi menonjol |
| Primary action | Indigo gelap, kandidat awal | Aksi paling penting per area tugas; bukan indikator berhasil |
| Text | Slate gelap; secondary tetap terbaca | Nilai penting tidak dibuat abu-abu terlalu pucat |
| Success | Hijau + teks/icon | Hasil yang benar-benar dikonfirmasi; tag katalog tidak membuktikan sertifikat aktif |
| Warning | Amber dengan teks gelap | Segera kedaluwarsa, antrean tertunda atau kondisi perlu perhatian |
| Danger/error | Merah + pesan spesifik | Penolakan, gagal validasi, aksi pembatalan/reversal yang berdampak |
| Information | Biru + label | Informasi kontekstual yang tidak mengharuskan tindakan segera |
| Neutral/pending | Slate + label status | Draft atau data belum diketahui; jangan pakai hijau untuk pending |
| Focus/selection | Token tersendiri | Bedakan item dipilih, hover, keyboard focus dan status bisnis |

Keputusan warna dipisah menjadi tiga tingkat:

```text
Primitive: palette.indigo.700 = warna dasar
Semantic:  color.action.primary = warna aksi utama
Component: button.primary.background = peran yang digunakan Button
```

Semantic token memungkinkan penggantian branding tanpa mengganti setiap komponen. `on-primary` berarti warna teks/icon di atas primary. `container` biasanya background lebih lembut untuk area status; pasangan teksnya perlu dicek tersendiri.

### Contoh pasangan palette light

| Peran | Foreground | Background | Rasio kontras terhitung |
|---|---|---|---:|
| Tombol utama | `#FFFFFF` | `#4338CA` | 7.90:1 |
| Teks utama | `#0F172A` | `#FFFFFF` | 17.85:1 |
| Teks secondary | `#475569` | `#FFFFFF` | 7.58:1 |
| Status sukses | `#166534` | `#F0FDF4` | 6.81:1 |
| Peringatan | `#92400E` | `#FFFBEB` | 6.84:1 |
| Error | `#991B1B` | `#FEF2F2` | 7.60:1 |
| Informasi | `#1E40AF` | `#EFF6FF` | 8.01:1 |

Dihitung pada 2026-09-05 dengan rumus luminansi relatif sRGB WCAG, untuk warna solid tanpa opacity. Pasangan ini contoh awal yang dapat diuji di prototype. Hasilnya tidak membuktikan seluruh UI accessible: hover, disabled, focus, border, overlay, gambar dan dark theme belum diuji. Foreground yang lolos pada putih belum tentu lolos di surface lain.

Wajib: tidak menjelaskan status hanya lewat warna; tidak meletakkan teks kuning terang di atas putih; tidak membuat semua badge hijau; tidak memakai merah untuk setiap tindakan agar semuanya terlihat penting. Rasio 60/30/10 boleh menjadi inspirasi komposisi, bukan acceptance criterion aplikasi operasional.

Light theme menjadi baseline awal. Dark theme menunggu seluruh pasangan semantic/state dipetakan dan diuji; jangan sekadar membalik warna atau membiarkan OS menghasilkan tema yang belum didesain. Branding tenant nantinya tidak boleh mengubah arti error/success atau menghilangkan kontras.

## 6. Tipografi, icon, spacing dan motion

Default ukuran berikut adalah pilihan proyek, bukan kewajiban Material/WCAG. Dalam CSS gunakan `rem`, relative line-height dan layout yang dapat membesar mengikuti teks.

| Peran | Default awal (ekuivalen pada root 16px) | Penggunaan |
|---|---|---|
| Body | 16px / line-height 1.5 | Form dan instruksi utama |
| Compact data/label | 14px / 1.4–1.5 | Tabel desktop, label; bukan alasan mengecilkan seluruh app |
| Metadata | 12–14px / 1.5 | Informasi tambahan; jumlah uang/risiko penting tetap lebih besar |
| Section title | 20px / 1.3 | Kelompok informasi |
| Page title | 24–28px / 1.2–1.3 | Judul layar |
| POS total | 28–32px / 1.2 | Jumlah yang akan dibayar |

- Gunakan satu keluarga sans-serif utama; Geist yang sudah ada dapat dipertahankan setelah lisensi aset diperiksa. Weight umum 400/500/600 cukup untuk awal.
- Angka uang memakai tabular numerals dan rata kanan pada tabel. Label akun/nama produk rata kiri. Debit/credit dan nilai negatif diberi label/tanda, bukan sekadar warna.
- Pakai sentence case: `Tambah produk`, bukan semua label HURUF BESAR. Hindari paragraph terlalu lebar; petunjuk panjang sekitar 60–75 karakter per baris sebagai titik awal.
- Skala jarak awal berbasis 4px: 4, 8, 12, 16, 24, 32, 48. Default label-input 8px, antarfield 16px, antarseksi 24–32px. Keterkaitan konten menentukan jarak; jangan mengikuti angka secara kaku bila merusak tugas.
- Radius awal 4/8/12px untuk kebutuhan berbeda; radius pill untuk badge, bukan semua container. Shadow menunjukkan layer/overlay, bukan hiasan setiap kartu.
- Gunakan satu icon family dan gaya stroke konsisten. Icon dekoratif disembunyikan dari accessibility tree; tombol icon-only wajib accessible name dan target klik memadai. Tooltip tidak menggantikan label penting.
- Motion ringan sekitar 100–200ms untuk feedback opsional; hormati reduced-motion dan hindari animasi checkout yang menunda kerja. Perubahan status transaksi harus tetap dapat dipahami tanpa animasi.

## 7. Layout dan responsive behavior

Mulai dari urutan membaca: identitas konteks → judul/tugas → isi → aksi/hasil. Satu area tugas biasanya memiliki satu aksi primer; aksi sekunder tidak perlu bersaing dengan warna dan ukuran yang sama.

Default operations layout: navigasi tenant/role, header berisi judul dan aksi, area filter, lalu tabel atau detail. Form panjang memakai satu kolom utama; dua kolom hanya untuk field pendek yang memang berkaitan. Hindari menaruh tabel lebar di kartu sempit atau tabel di dalam tabel.

Default POS layout pada desktop: area katalog fleksibel dan panel keranjang sekitar 360–420px jika viewport cukup. Ketika sempit, gunakan perpindahan katalog/keranjang yang eksplisit atau susun vertikal; jangan mempertahankan dua panel sempit dengan tulisan kecil. Tombol total/bayar tetap terjangkau dan tidak menutup baris terakhir atau keyboard focus.

Grid awal dapat memakai 12 kolom desktop, 8 tablet dan 4 compact; jumlah ini alat bantu, bukan aturan universal. Breakpoint dipilih saat isi tidak lagi muat, bukan hanya berdasarkan nama perangkat. Baseline review: 1440px desktop, 1024px landscape, 768px compact dan 390px phone, ditambah pengujian reflow aksesibilitas.

Wajib uji nama produk panjang, terjemahan lebih panjang, cart 1/20 item, jumlah besar, empty state dan zoom. Jangan mengunci tinggi input/card yang memotong teks. Bila ada scroll pada katalog dan keranjang, jelas batas dan focus-nya; hindari banyak scroll container bersarang.

Laptop, tablet dan HP menjadi target operasional Phase 3 sesuai perluasan kebutuhan 2026-09-05. Paket [POS responsive](design/screens/POS_RESPONSIVE.md) menetapkan adaptasi awal; feature lain tetap membutuhkan review lintas perangkat. Tabel dua dimensi dapat memiliki horizontal scroll yang jelas tanpa membuat seluruh halaman ikut meluber; POS compact mengutamakan blok item berlabel. Layout receipt dicatat terpisah dari layout layar, termasuk printer 80mm dan kondisi gagal cetak. Dukungan browser/printer tidak otomatis terbukti oleh wireframe.

## 8. Pola halaman yang dipakai ulang

| Pola | Anatomi standar | State penting |
|---|---|---|
| List/report | Judul, filter, hasil/summary, tabel, navigasi hasil | Loading, kosong belum ada data, kosong karena filter, gagal, stale |
| Detail | Identitas/status, ringkasan, data/baris, sumber dokumen, aksi sesuai permission | Tidak ditemukan, forbidden, data berubah, efek aksi |
| Create/edit | Judul, field terkelompok, helper/error, ringkasan, batal/simpan | Pristine, dirty, invalid, pending, gagal, sukses |
| Confirmation | Aksi spesifik, objek/jumlah/dampak, alasan bila kontrak mendukung, tombol jelas | Pending, konflik, sukses; close tidak berarti membatalkan request server |
| POS workspace | Search/scan, katalog, cart, totals, payment, koneksi | Katalog stale, stok habis, cart kosong, nominal salah, outcome unknown |
| Recovery center | Jumlah tertunda, waktu/status, detail masalah, tindakan pemulihan | Auth required, retry wait, perlu review, confirmed |
| Financial editor | Akun/baris, debit/credit, difference, memo, review/post | Tidak seimbang, akun invalid, pending, posted/reversed |

Pola lebih penting daripada menyalin seluruh template halaman. Form supplier dan produk dapat berbagi Field/ActionBar tanpa memaksa model domain yang sama. Pisahkan primitives dari domain: `Button` tidak perlu mengetahui PO; `CertificateStatus` boleh mengetahui status sertifikat yang didukung API.

Setiap komponen shared mencatat purpose, kapan digunakan/tidak digunakan, props/variants, keyboard/accessibility contract, loading/error/disabled behavior dan contoh nyata. Default Button cukup primary/secondary/ghost/destructive serta ukuran compact/default/touch; tambah variant jika ada kebutuhan berulang yang tidak tertangani.

## 9. Form, tabel dan microcopy

### Form

- Label tetap terlihat; placeholder hanya contoh. Tulis satuan dan format sebelum pengguna mengetik. Jelaskan field wajib/opsional secara konsisten.
- Validasi awal pada blur/submit bila masuk akal; jangan menampilkan error keras sebelum field disentuh. Setelah error muncul, perbarui feedback ketika pengguna memperbaikinya.
- Tampilkan pesan di field terkait serta ringkasan jika banyak error. Focus menuju error pertama/ringkasan setelah submit gagal, tanpa menghilangkan isi.
- Tombol submit boleh tetap aktif untuk menampilkan validasi yang membantu. Disable bila invariant jelas tidak terpenuhi (misalnya debit ≠ credit), permission tidak ada, atau request sedang berjalan; alasannya harus terlihat tanpa bergantung pada hover disabled button.
- Mencegah klik ganda di UI tidak menggantikan idempotency backend. Jangan menganggap menutup dialog membatalkan transaksi yang sudah dikirim.
- Dirty form memerlukan penanganan navigasi keluar. Undo hanya ditampilkan jika operasi benar-benar dapat dibatalkan sesuai kontrak.

### Tabel/report

- Teks rata kiri, nilai numerik rata kanan, tanggal/format konsisten. Gunakan header dan caption/label yang menjelaskan isi; `table` biasa menjadi default, bukan ARIA grid kecuali keyboard grid benar-benar diimplementasikan.
- Search, sorting, filter dan pagination harus sesuai kemampuan API. Jika filter hanya berlaku pada data yang telah dimuat, nyatakan cakupannya; jangan menyajikannya sebagai pencarian seluruh tenant.
- Jangan membuat total page/record palsu ketika API hanya menyediakan panjang halaman. Simpan filter non-sensitif di URL bila membantu navigasi; jangan memasukkan nama pelanggan atau token ke URL.
- Truncation tidak boleh menyembunyikan informasi finansial/compliance penting. Sediakan detail yang dapat dibuka dengan keyboard/touch, bukan tooltip hover saja.
- Bedakan nilai `0` yang nyata dari data tidak tersedia. Tampilkan waktu/periode laporan dan kondisi data stale. Chart harus menyertakan label/legend atau tabel alternatif; kategori chart tidak memakai warna status secara sembarang.

### Microcopy

| Hindari | Gunakan |
|---|---|
| `Error 422` saja | `PO tidak dapat dibuat: sertifikat pemasok sudah kedaluwarsa.` |
| `Success` untuk offline | `Tersimpan di perangkat. Menunggu sinkronisasi.` |
| `Apakah Anda yakin?` | `Batalkan transaksi TXN-…? Stok dan jurnal transaksi akan dibalik.` |
| `Gagal. Coba lagi.` untuk timeout | `Hasil transaksi belum diketahui. Periksa status sebelum membuat transaksi baru.` |

Pesan menyebut masalah, dampak dan langkah berikutnya yang benar-benar tersedia. Simpan reference/trace ID sebagai detail bantuan; jangan menampilkan stack trace, SQL atau key teknis sebagai instruksi kasir. Label internal seperti `Idempotency-Key` tidak diperlukan pada alur pembayaran.

## 10. Aksesibilitas sebagai acceptance

Target proyek: WCAG 2.2 AA. Batas di bawah tidak merupakan daftar lengkap; audit tetap meliputi alur nyata.

| Area | Kriteria yang diperiksa |
|---|---|
| Kontras teks | Minimal 4.5:1 teks biasa; 3:1 untuk large text menurut definisi WCAG (18pt normal atau 14pt bold). Jangan menganggap semua heading otomatis large |
| Non-text | Informasi visual yang diperlukan untuk mengenali kontrol/state dan bagian grafik penting memenuhi 3:1 terhadap warna bersebelahan, dengan pengecualian standar |
| Target | WCAG AA menetapkan 24×24 CSS px atau pengecualian seperti spacing. Default internal Tenet untuk POS touch adalah 44–48px; ini pilihan ergonomi yang lebih besar, bukan angka minimum AA |
| Keyboard/focus | Semua tindakan tersedia via keyboard, urutan logis, focus terlihat dan tidak tertutup sticky layer; jangan hapus outline tanpa pengganti |
| Reflow | Uji konten pada lebar ekuivalen 320 CSS px; pengecualian berlaku untuk konten yang memang memerlukan layout dua dimensi, bukan seluruh halaman |
| Form/status | Label terhubung, error dijelaskan dalam teks, status penting diumumkan secara tepat; jangan membuat semua perubahan cart menjadi alert yang mengganggu |

Rujukan: [text contrast](https://www.w3.org/WAI/WCAG22/Understanding/contrast-minimum.html), [non-text contrast](https://www.w3.org/WAI/WCAG22/Understanding/non-text-contrast.html), [target size](https://www.w3.org/WAI/WCAG22/Understanding/target-size-minimum.html), [reflow](https://www.w3.org/WAI/WCAG22/Understanding/reflow.html).

Dialog modal memiliki nama yang jelas, focus awal yang masuk akal, focus tetap di dalam selama modal aktif, dan kembali ke pemicu/tujuan logis setelah ditutup. Escape umumnya menutup dialog; bila suatu proses membatasi dismissal, perilakunya dijelaskan dan tidak boleh menjebak pengguna. [WAI dialog pattern](https://www.w3.org/WAI/ARIA/apg/patterns/dialog-modal/)

Gunakan native HTML sebelum menambah ARIA. Primitive accessible membantu focus/keyboard, tetapi label, susunan halaman dan modifikasi komponen tetap tanggung jawab aplikasi. [Radix accessibility](https://www.radix-ui.com/primitives/docs/overview/accessibility)

Pemeriksaan otomatis tidak membuktikan kepatuhan menyeluruh. Uji keyboard, screen reader, zoom 200%, reflow dan reduced motion. Jangan menyebut UI compliant hanya karena score Lighthouse/axe bagus. Persyaratan Focus Appearance 2.4.13 berada pada AAA, bukan AA; jika dijadikan target internal tambahan, tandai demikian. [WCAG 2.2 levels](https://www.w3.org/WAI/standards-guidelines/wcag/new-in-22/)

## 11. State domain yang tidak boleh terlewat

- Kasir: empty cart, scan tidak ditemukan, stok/harga stale, tender kurang, pending, outcome unknown, confirmed dan print gagal. Gagal cetak tidak membuat checkout baru.
- Offline: session kedaluwarsa, quota storage penuh, queue tersimpan, reconnect, konflik bisnis, retry terjadwal, multi-tab dan recovery setelah restart. Logout tidak otomatis menghapus transaksi berbayar yang belum sinkron.
- Compliance: tag katalog, sertifikat valid, belum berlaku, segera kedaluwarsa, kedaluwarsa, hilang dan dicabut perlu dibedakan hanya sejauh data API memungkinkan. Catat gap jika backend menggabungkan status; jangan mengarang history atau tombol override strict mode.
- Supply chain: ordered/received/remaining, partial receipt, PO dibatalkan dan sertifikat berubah sejak pemesanan.
- Ledger: total debit/credit/difference, akun invalid, posted, reversal; jangan menyediakan edit pada posted entry atau menjanjikan undo yang tidak didukung.
- Permission: hide aksi yang tidak relevan secara permanen; disabled dengan alasan untuk aksi sementara tidak tersedia. Forbidden dari server tetap ditangani walaupun tombol tersembunyi.

Semua nilai demo pada wireframe harus diberi konteks contoh dan berasal dari data sintetis yang aman. Data contoh dalam artefak desain bukan izin membuat endpoint palsu, mengganti hasil API dengan angka statis, atau menghilangkan tes.

## 12. Workflow feature dari awal sampai selesai

| Tahap | Hasil kecil yang diperiksa | Syarat lanjut |
|---|---|---|
| 1. Pahami tugas | Role, tujuan, input/output dan risiko satu UX ID | Tugas dan kemampuan API jelas; gap ditandai |
| 2. Petakan flow/state | Happy path + failure penting | Tidak ada sukses palsu atau aksi recovery yang belum tersedia |
| 3. Wireframe | Struktur satu viewport dengan isi realistis | Urutan informasi dan focus dapat dijelaskan |
| 4. Terapkan sistem | Tokens dan komponen yang sudah ada | Variant baru punya alasan; palette/layout tidak berubah tanpa keputusan |
| 5. Prototype/review | Uji tugas, tablet/keyboard dan pesan error | Catat observasi; role simulation tidak diklaim user research |
| 6. Spesifikasi siap implementasi | Kontrak layar dari template, acceptance dan dependency | State/permission/data diketahui atau feature dibatasi secara eksplisit |
| 7. Implementasi vertikal | Komponen, API nyata, validasi, storage bila perlu | Perilaku lengkap untuk scope kecil, tidak hanya happy path |
| 8. Verifikasi | Test behavior, visual/state review, aksesibilitas | Temuan penting diselesaikan, bukti hasil dicatat |
| 9. Arsip dan review perubahan | Spec/token/source/export konsisten, changelog bila perlu | Portabilitas dijaga; commit/publikasi mengikuti governance |

Perubahan field sederhana boleh memakai review ringan dalam satu sesi. Alur cash offline/auth/ledger membutuhkan pemeriksaan domain dan backend lebih dalam. Jangan mewajibkan dokumen panjang, diagram baru atau custom abstraction untuk setiap penyesuaian kecil.

Definition of Ready: role/tugas, kontrak API, pola komponen, state, acceptance examples dan sumber desain diketahui. Definition of Done: perilaku terverifikasi, desain/implementasi konsisten, tidak ada state penting yang hilang, test dan dokumentasi terkait mutakhir.

Perintah wajib sebelum completion mengikuti repository: backend `go build ./... && go vet ./... && go test -race -short ./...`; frontend `npm run lint && npm run build`. Tambahkan tes feature/visual/device yang relevan ketika tersedia. Tidak perlu tes yang hanya menyalin implementasi, tetapi jangan menghapus/mengabaikan tes gagal untuk mengklaim selesai.

## 13. Konsistensi dan pemeliharaan jangka panjang

- Satu sumber semantic tokens; CSS dan editor menggunakan turunannya. Jangan menulis hex/spacing baru berulang di feature bila token yang tepat sudah ada.
- Kode bersama memiliki variants yang jelas dan contoh state. Hindari membuat komponen berbeda untuk tiap halaman yang hanya berbeda warna.
- Perubahan warna/status/keyboard yang berdampak lintas layar harus meninjau semua pemakai. Catat alasan dan contoh sebelum/sesudah pada design changelog; preview yang tidak berubah adalah bukti penting juga.
- Catat versi design-system baseline dan revision tiap layar. Perubahan props atau arti token yang mematahkan pemakai membutuhkan migration note/deprecation; gunakan alias sementara jika migrasi tidak sekaligus. Versi ini terpisah dari release aplikasi.
- Simpan sumber Penpot beserta library, SVG preview, token JSON dan state spec. Native export diuji restore; import SVG ke editor lain dapat kehilangan variants/interaction.
- Evaluasi dependency dan lisensi saat adopsi/upgrade. Jangan menganggap kit gratis selalu boleh didistribusikan ulang atau fitur premium bagian dari baseline.
- Review drift setiap milestone: apakah ada lima warna sukses berbeda, dialog yang focus-nya berbeda, atau format rupiah yang tidak sama? Perbaiki lewat token/komponen pusat dengan regression review.
- Default awal: satu light theme, satu arah visual, dua shell, dan sedikit pola yang teruji. Multi-brand, dark mode, drag-and-drop dashboard builder dan microfrontend ditunda sampai kebutuhan nyata membenarkan biaya.

## 14. Aturan kerja lintas AI dan template

AI dari perusahaan mana pun mengikuti pedoman yang sama. Untuk tugas kecil, berikan UX ID, bagian kaidah terkait, sumber komponen dan acceptance case; minta satu perubahan terfokus. Jangan meminta setiap model mendefinisikan ulang branding, struktur folder atau seluruh design system.

Setelah tugas: catat file/keputusan, hasil tes sebenarnya, status verifikasi dan langkah berikutnya. Catatan sesi/quota tetap lokal. Reviewer menilai hasil, bukan nama vendor/model. Adapter editor opsional boleh membantu membaca pedoman, tetapi sumber kaidah adalah Markdown publik ini.

Gunakan [template spesifikasi layar](design/SCREEN_SPEC_TEMPLATE.md) untuk flow baru. Checklist review singkat:

- [ ] Tugas/role dan endpoint yang digunakan benar.
- [ ] Struktur, istilah, tokens dan komponen konsisten.
- [ ] Default/loading/empty/error/permission/offline terkait dijelaskan.
- [ ] Warna memiliki label; pasangan kontras dan keyboard/focus diuji.
- [ ] Data panjang, angka besar, small viewport dan zoom ditinjau.
- [ ] Request pending/unknown dan destructive action ditangani benar.
- [ ] Bukti test, file sumber dan keterbatasan tercatat.

Pedoman dapat berkembang melalui perubahan kecil yang beralasan. Aturan baru harus menyelesaikan masalah berulang atau risiko nyata, bukan sekadar menambah proses.

---

## 15. Aturan Disiplin Multi-Bahasa (i18n) dan Lokalisasi

Tanggal: 2026-09-05. Wajib dipatuhi untuk seluruh implementasi halaman baru (inventory, procurement, ledger, manager dashboard) dan modifikasi komponen FE.

Tenet Commerce dirancang sebagai platform ritel multi-tenant berstandar enterprise internasional yang melayani operasional ritel syariah. Oleh karena itu, antarmuka mendukung tiga bahasa secara simultan: **Bahasa Indonesia (`id`)**, **English (`en`)**, dan **العربية / Arabic (`ar`)**.

### 15.1 Prinsip Nol Hardcoded String (Zero Hardcoded String)

1. **Larangan Teks Statis:**
   - Dilarang keras menulis string bahasa manusia yang terlihat di antarmuka (*UI text*) langsung di dalam file `.tsx` atau `.jsx`.
   - Meliputi: judul halaman, label input, placeholder form, teks tombol CTA, pesan validasi, toast notification, header dan sel tabel, modal konfirmasi, teks badge status, hingga deskripsi *empty-state*.
   - Semua teks wajib dipanggil menggunakan hook `useTranslation()`:
     ```tsx
     // ❌ SALAH (Hardcoded string):
     <button>Tambah Produk</button>
     <input placeholder="Cari nama barang..." />

     // ✅ BENAR (Menggunakan i18n key):
     const { t } = useTranslation();
     <button>{t("inventory.actions.addProduct")}</button>
     <input placeholder={t("inventory.searchPlaceholder")} />
     ```

2. **Paritas Kunci 1:1 (Key Parity Invariant):**
   - Setiap kali sebuah string baru ditambahkan atau diubah, kunci tersebut **wajib** dideklarasikan secara serentak pada 3 file kamus:
     - `frontend/src/lib/i18n/locales/id.ts` (Bahasa Indonesia)
     - `frontend/src/lib/i18n/locales/en.ts` (English)
     - `frontend/src/lib/i18n/locales/ar.ts` (Arabic)
   - Dan tipe skemanya wajib terdaftar di `frontend/src/lib/i18n/types.ts` (`TranslationSchema`).
   - Tidak boleh ada kunci yang ada di `id` tetapi terlewat di `en` atau `ar`.

### 15.2 Konvensi Penamaan dan Struktur Hirarki Kunci

Gunakan notasi camelCase bertingkat yang mencerminkan domain dan komponen:

```text
<domain>.<subdomain-atau-komponen>.<namaElemen>
```

Contoh domain yang terdaftar:
- `common`: Tindakan global (`actions.save`, `actions.cancel`, `actions.delete`), status umum (`status.active`, `status.inactive`), konfirmasi, indikator jaringan.
- `nav`: Label navigasi pada sidebar dan menu header.
- `auth`: Layar otentikasi login, input tenant/email/password, error kredensial.
- `pos`: Register kasir, katalog, pencarian, pill kategori, badge sertifikasi halal.
- `tender`: Modal pembayaran tunai, kembalian, preset nominal pas.
- `receipt`: Struk transaksi pembayaran, preview modal, struk thermal print.
- `history`: Riwayat transaksi penjualan, filter tanggal, modal pembatalan (void).
- `inventory`: Daftar produk, kategori, form tambah/edit produk, penyesuaian stok opname, peringatan stok menipis.
- `supplyChain`: Sertifikasi halal pemasok, pesanan pembelian (PO), penerimaan barang (GR).
- `ledger`: Jurnal debit/kredit, bagan akun (COA), neraca saldo.

### 15.3 Kosakata Syariah dan Tata Bahasa Arab Baku

Penerjemahan ke Bahasa Arab (`ar`) bukan sekadar alih bahasa mesin, melainkan menggunakan terminologi perbankan, akuntansi, dan ritel syariah yang baku:
- Produk Bersertifikat Halal: **منتج معتمد حلال**
- Double-Entry Ledger: **دفتر الأستاذ ذو القيد المزدوج**
- Subtotal: **المجموع الفرعي**
- Pajak / PPN: **الضريبة (0%)**
- Total Tagihan: **الإجمالي الكلي**
- Tunai Diterima: **المبلغ النقدي المستلم**
- Uang Kembalian: **المبلغ المتبقي / الباقي**
- Pembatalan Transaksi (Void): **فسخ الفاتورة / إلغاء المعاملة**
- Stok Opname / Penyesuaian: **جرد المخزون / تسوية البضائع**

### 15.4 Dukungan Arah Teks (RTL - Right-to-Left)

1. **Arah Dokumen Otomatis:**
   - Konteks `I18nProvider` secara otomatis mengatur atribut HTML `document.documentElement.dir = "rtl"` saat bahasa `ar` dipilih, dan `dir = "ltr"` untuk `id` dan `en`.
2. **Kaidah Styling Bidirectional:**
   - Hindari penggunaan hardcoded margin/padding absolute seperti `left-0` atau `right-0` jika elemen tersebut merupakan ikon yang posisinya harus berpindah saat RTL.
   - Gunakan logical properties Tailwind atau varian `rtl:` jika diperlukan:
     - Gunakan `start` dan `end` alih-alih `left` dan `right` untuk perataan teks (`text-start`, `text-end`).
     - Ikon panah navigasi maju/mundur (`ArrowRight` / `ArrowLeft`) disesuaikan orientasinya secara kontekstual saat RTL.

### 15.5 Interpolasi Variabel Dinamis

Dilarang melakukan konkatenasi string manual (misal: `"Sisa stok: " + qty + " item"`).
Gunakan parameter kurung kurawal `{param}` yang didukung oleh interpolator `translate()`:

```ts
// types.ts
remainingStock: string; // e.g. "Sisa {count} item"

// id.ts: remainingStock: "Sisa {count} item"
// en.ts: remainingStock: "{count} items remaining"
// ar.ts: remainingStock: "متبقي {count} عناصر"

// Penggunaan di komponen:
t("inventory.remainingStock", { count: product.stock_quantity })
```

### 15.6 Format Angka, Uang, dan Tanggal

1. **Nominal Rupiah (IDR):**
   - Gunakan pustaka pembantu terpusat di `frontend/src/lib/money.ts` (`formatIDR`, `parseIDR`).
   - Jangan menulis simbol `Rp` atau `.000` secara manual di dalam kamus bahasa. Simbol dan pemisah ribuan ditangani secara konsisten oleh helper moneter.
2. **Waktu dan Tanggal:**
   - Gunakan pustaka pembantu di `frontend/src/lib/date.ts` (`formatDateTime`) yang menghasilkan representasi waktu yang rapi dan konsisten.

### 15.7 Alur Kerja Wajib (Developer Workflow) Saat Menambah Halaman/Komponen Baru

Setiap developer atau AI agent yang mengimplementasikan fitur UI baru wajib mengikuti 6 langkah disiplin berikut:

1. **Step 1 — Perencanaan Kunci:** Catat seluruh teks yang akan ditampilkan pada layar (heading, button, column header, error message).
2. **Step 2 — Daftarkan Skema:** Tambahkan struktur kunci baru ke `frontend/src/lib/i18n/types.ts` di bawah domain terkait.
3. **Step 3 — Isi Kamus Bahasa:** Masukkan teks terjemahan ke `locales/id.ts`, `locales/en.ts`, dan `locales/ar.ts` secara lengkap.
4. **Step 4 — Pasang di Komponen:** Panggil teks menggunakan hook `const { t } = useTranslation();`.
5. **Step 5 — Uji Paritas Kunci Otomatis:**
   ```bash
   npm test
   ```
   Pastikan pengujian `i18n: dictionary key parity across id, en, and ar` lulus 100% tanpa ada kunci yang hilang (*missing key*).
6. **Step 6 — Verifikasi Tampilan Visual:**
   Periksa di browser bahwa teks berganti secara instan saat beralih antara 🇮🇩 ID, 🇬🇧 EN, dan 🇸🇦 AR, serta tata letak RTL tidak mengalami kerusakan visual.
