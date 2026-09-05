# Tenet design artifacts

Package revision `pos-responsive-0.2`, 2026-09-05. Status: **proposed, not implementation-ready**. Desktop source tetap `pos-lowfi-0.1`; delapan varian tablet/HP baru v0.2. Review struktur dan keselamatan transaksi dahulu, kemudian visual detail.

## Paket POS tunai online

| Artefak | Isi |
|---|---|
| [Peta kontrak](POS_CONTRACT_MAP.md) | API yang benar-benar tersedia, batas dan dependency |
| [Alur kasir](flows/pos-cash-online.mmd) | Happy path, penolakan dan hasil belum diketahui |
| [Spesifikasi tiga layar](screens/POS_CASH_ONLINE.md) | Data, interaksi, state, acceptance dan handoff |
| [POS-01: katalog dan keranjang](exports/pos-01-workspace.svg) | Cari produk → periksa keranjang |
| [POS-02: pembayaran](exports/pos-02-payment.svg) | Periksa total → isi uang diterima → kirim sekali |
| [POS-03: hasil transaksi](exports/pos-03-result.svg) | Bukti server, kembalian dan cetak ulang |

## Paket responsive v0.2

Aturan dan acceptance lengkap: [POS responsive — laptop, tablet, HP](screens/POS_RESPONSIVE.md).

| Layar | Tablet 768×1024 | HP 390×844 |
|---|---|---|
| Katalog dan cart | [Workspace tablet](exports/pos-01-tablet-workspace.svg) | [Katalog HP](exports/pos-01-phone-catalog.svg) → [Keranjang HP](exports/pos-01-phone-cart.svg) |
| Pembayaran | [Pembayaran tablet](exports/pos-02-tablet-payment.svg) | [Pembayaran HP](exports/pos-02-phone-payment.svg) |
| Hasil confirmed | [Hasil tablet](exports/pos-03-tablet-result.svg) | [Hasil HP](exports/pos-03-phone-result.svg) |
| Hasil belum diketahui | Perilaku dijelaskan dalam spec | [Unknown HP](exports/pos-03-phone-unknown.svg) |

SVG adalah sumber vektor yang dapat diedit, bukan screenshot maupun prototype interaktif. Buka langsung di browser tanpa akun atau koneksi. Paket berisi 11 frame: tiga desktop, tiga tablet dan lima HP, memakai satu arah grayscale. Semua identitas/produk/angka bertanda contoh merupakan data sintetis; bukan hasil API atau transaksi nyata. Tinggi frame adalah contoh viewport, bukan fixed-height aplikasi.

## Manifest dan portabilitas

- Sumber kanonik tahap ini: SVG + Markdown + Mermaid. Desktop/kontrak/flow awal tetap v0.1; tambahan responsive v0.2 mewarisi kontrak dan aturan transaksi v0.1. Belum ada sumber `.penpot`, native components, Auto Layout, interactions atau token JSON final.
- Dibuat sebagai teks/vector lokal; tanpa plugin, layanan AI runtime, gambar eksternal, font unduhan atau asset kit pihak ketiga. Font: system `sans-serif`; fallback dapat mengubah metrik teks. Tidak membundel font. Distribusi mengikuti lisensi repository.
- Setiap SVG memiliki `title`, `desc` dan ID region. SVG tidak membuktikan aksesibilitas aplikasi interaktif.
- Impor visual ke editor mengikuti [batas format dalam rencana Phase 3](../FRONTEND_PHASE3_DESIGN.md). Figma/Penpot import belum diuji dalam sesi ini; jangan menganggap groups otomatis menjadi komponen. Miro bukan sumber layout kanonik.
- Setelah review: impor ke editor pilihan, periksa teks/posisi/groups, buat interaksi bila diperlukan, lalu ekspor native beserta library dan uji restore. Catat kehilangan format di dokumen ini. Tidak perlu akun berbayar untuk membaca/review paket lokal.
- Jika mengedit di editor, perbarui SVG dan spec bersama-sama; jangan menjadikan tautan cloud satu-satunya salinan.

## Pemeriksaan revision 0.1

- Ketiga SVG valid XML (`xmllint`) dan telah dirender dengan macOS Quick Look lalu diperiksa visual: nama, angka, panel kanan dan tombol terlihat tanpa terpotong pada frame contoh. Preview memakai wrapper kanvas persegi sementara karena thumbnail Quick Look awal memotong aspek rasio landscape; SVG sumber tetap 1440×900.
- Backend `go build ./... && go vet ./... && go test -race -short ./...`: exit 0, hasil paket test cached. Frontend `npm run lint && npm run build`: exit 0 pada scaffold existing. Ini bukan bukti acceptance checkout/E2E desain.
- Mermaid disediakan sebagai source; rendering diagram belum diuji. Native editor import/restore, responsive prototype, keyboard/screen reader dan printer belum diuji.

## Pemeriksaan responsive v0.2

- Delapan SVG baru dirender dalam empat lembar review lokal dan diperiksa visual: teks contoh, total, kontrol jumlah, navigasi compact dan state unknown terbaca tanpa overlap yang terlihat. Ini pemeriksaan statis pada 768×1024 dan 390×844, bukan tes CSS responsive atau perangkat nyata.
- Seluruh 11 SVG lolos XML validation, pemeriksaan ID unik per file, title/desc, dan tidak memiliki script/foreignObject atau referensi asset HTTP. Sebanyak 46 tautan lokal dalam Markdown paket valid pada pemeriksaan ini; code fence berpasangan. `git diff --check` lulus.
- Backend build/vet/race-short dan frontend lint/build kembali exit 0; paket Go memakai cached test results, FE tetap scaffold. Tidak ada tes interaksi baru yang diklaim lulus.
- Belum diuji: reflow 320, landscape/1024, keyboard layar, screen reader, lifecycle mobile, perangkat/printer nyata, native import/restore dan prototype interaktif. Lihat R-01..R-10 sebelum menerima implementasi.

## Review pertama — 15–20 menit, estimasi

1. Buka POS-01: tunjukkan cara membeli dua susu dan satu roti. Apakah SKU, jumlah dan total mudah ditemukan?
2. Buka POS-02: uang Rp50.000 untuk total Rp40.000. Apakah tombol menyatakan dampak yang jelas?
3. Buka POS-03: apakah transaksi berhasil tetap jelas ketika cetak tidak keluar?
4. Baca state `unknown` di spec: bisakah kasir menjelaskan mengapa tidak boleh menagih ulang?

Catat kesulitan dan keputusan, bukan hanya “bagus/tidak bagus”. Review pemilik proyek bukan riset kasir independen. Persetujuan layout tidak menutup dependency backend.

Untuk review responsive, ulangi tugas yang sama pada tablet/HP, termasuk kembali dari pembayaran ke keranjang sebelum submit. Nilai kemudahan navigasi dan keterbacaan; jangan menyebut aplikasi mobile-friendly hanya karena SVG dapat diperkecil.
