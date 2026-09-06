# Review implementasi FE dan backlog responsive

Tanggal: 2026-09-05. Status: **review sumber dan rencana perbaikan; bukan persetujuan mobile atau release**.

Baseline: commit `0e836ae` beserta perubahan kerja i18n yang ada saat review. Temuan pada header dan terjemahan mencakup perubahan yang belum di-commit. Tidak ada pengujian browser otomatis dalam review ini. Dampak visual di bawah merupakan analisis struktur CSS/React, bukan hasil pengukuran viewport atau perangkat nyata.

Rujukan: [pedoman FE](../FRONTEND_GUIDELINES.md), [Phase 3](../FRONTEND_PHASE3_DESIGN.md), [acceptance responsive R-01–R-10](screens/POS_RESPONSIVE.md), dan [dependency kontrak](POS_CONTRACT_MAP.md).

## Tindak lanjut implementasi: FE-S1, bagian pertama

### Pembaruan 2026-09-06 — pemilih bahasa login dan header

Revisi preferensi pengguna: selector berikut diperkecil menjadi native dropdown kode `ID / EN / AR`, lebar 80px dan area sentuh 44px, tanpa border/background dominan. Nama bahasa asli tetap menjadi accessible label pada opsi; kode Latin berarah LTR. Header kembali satu baris dengan kolom identitas fleksibel dan kontrol ringkas, bukan baris khusus bahasa. Nama tenant panjang memakai ellipsis dengan teks lengkap tetap di DOM/title (identitas ini bukan data transaksi); login tetap document flow agar tidak overlap logo. Revisi ini menggantikan ukuran 48px, nama opsi panjang, dan header dua baris yang dijelaskan sebagai iterasi sebelumnya di bawah. Tidak mengubah locale provider atau fitur bisnis.

Menindaklanjuti laporan pengguna bahwa menu bahasa bertumpuk, source baseline `b48a8e8` menunjukkan selector login absolut di sudut kartu serta header satu baris dengan tinggi tetap dan segmented selector. Ini menggantikan deskripsi historis header di bawah apabila berbeda dari source baseline tersebut.

- Login: selector berada dalam document flow pada baris tersendiri sebelum logo, bukan overlay absolut. Header kartu mendapat jarak terpisah.
- Shared selector: satu native select dengan accessible name terjemahan, nama bahasa asli, arah opsi AR, font 16px dan target 48px. Varian compact/segmented yang tidak lagi digunakan dihapus; locale di luar registry tidak diteruskan ke setter. Provider/persistence locale tidak berubah.
- Header: tinggi mengikuti konten; di bawah 480px identitas dan baris bahasa/logout terpisah, mulai 480px memakai dua kolom dengan identitas yang dapat membungkus. Menu/logout mempunyai target 48px. Role tetap tersedia di sidebar. Badge koneksi statis dan fallback nama tenant fiktif dihapus; tidak menambah pemeriksaan koneksi atau mengubah auth.
- Dua tes tambahan memeriksa handler selector ID/EN/AR serta kontrak struktur layout login/header. Tes ini bukan pengukuran layout browser. Acceptance manual: 320/390/480/768px, tenant panjang, ID/EN/AR, keyboard dan zoom 200%; pastikan selector tidak menutup logo/identitas dan fokus tetap terlihat.

Verifikasi: 46 tes, FE lint/build pada salinan sementara, serta backend build/vet/race-short lulus (exit 0). Kesalahan path relatif pada tes baru di run awal diperbaiki sebelum run final; tidak ada tes dihapus atau dilewati. Tidak menjalankan browser otomatis atau mengganggu dev server.

Implementasi SaaS tetap ditunda. Perubahan ini merupakan patch lanjutan lokal, bukan bagian dari commit FE-R3 `b48a8e8`.

### Status terbaru 2026-09-06 — FE-R3 dialog dan receipt

Baseline kini `dfaefe1` (develop), termasuk merge shell/touch serta inventory. Catatan R1/R2 sebelumnya adalah riwayat, bukan pernyataan bahwa perubahan tersebut masih uncommitted. Patch dialog ini berada pada branch lokal `fix/52-pos-dialog-responsive`; belum commit/remote write.

- Primitive Modal memakai native `<dialog>`: nama/deskripsi terhubung, modal background inert dan focus containment/restoration mengikuti browser. Saat dibuka, scroll dialog kembali ke awal dan fokus menuju heading, bukan input. Tinggi dibatasi ruang `100dvh` dan isi memakai satu scroll dialog, bukan panel receipt 55vh yang bersarang. Consumer history turut memakai primitive ini dan perlu regression review manual.
- Dismissal dikontrol prop `dismissible`; pending/unknown tidak dapat ditutup dengan Escape, backdrop, atau tombol X. Klik di dalam area dialog/scrollbar tidak dianggap klik backdrop. Guard controller pembayaran sebelumnya tetap dipertahankan; tidak menambahkan retry atau key baru.
- Tender tidak autofocus dan menggunakan tinggi input yang benar-benar tersedia, logical spacing RTL, preset/aksi setidaknya 48px, serta aksi bertumpuk pada layar sempit. Label peringatan/bantuan baru lengkap ID/EN/AR. Pesan error yang berasal dari controller masih mengikuti implementasi FE-S1; patch ini bukan audit seluruh lokalisasi error domain.
- Receipt menempatkan status server, nomor transaksi, total dan kembalian sebelum tombol cetak/transaksi baru. Detail 20+ item menjadi disclosure yang dapat dibuka; nama/SKU/angka wrap. Semua jalur penutupan receipt tetap memakai lifecycle confirmed yang mengosongkan cart berbayar. Print CSS menyembunyikan dialog/backdrop agar tidak menutupi struktur thermal; hasil printer belum diverifikasi.

Finalisasi source FE-R3: label preset nominal khusus ID/EN/AR menggantikan label Filter; petunjuk shortcut `(Enter)` dihapus karena tidak ada global Enter handler untuk pembayaran. Deskripsi/error panjang dapat membungkus, bantuan unknown memakai warna semantic warning, dan disclosure referensi memakai target 48px. Tidak menambahkan shortcut pembayaran atau mengubah controller/domain transaksi.

Verifikasi terbaru: 44 tes lulus (termasuk inventory existing, dismissal modal/urutan receipt, effect setup/cleanup, guard tender pending/unknown, validasi nominal dan print callback tanpa sale baru). Frontend lint/build dan backend build/vet/race-short exit 0; build frontend memakai salinan sementara seperti dijelaskan di bawah. Tes menjalankan source/event handler/effect dengan adapter terbatas tanpa browser/DOM; tidak membuktikan interaksi native dialog, keyboard virtual, hasil print atau layout perangkat.

Build sebelumnya pada workspace gagal membaca chunk `.next` ketika dev server aktif. Verifikasi final dilakukan pada salinan frontend sementara dengan dependency existing, tanpa `.env*`, tanpa mengubah konfigurasi repo atau menghentikan dev server. Build salinan berhasil; ini mendukung dugaan benturan output/cache, bukan bukti forensik penyebab tunggal. Environment produksi dan E2E tetap belum diuji.

Acceptance manual FE-R3: buka/tutup dialog history dan pembayaran dengan keyboard; pastikan fokus kembali ke pemicu/logical destination; lihat dialog pada 320/390px, landscape dan zoom; isi nominal dengan keyboard layar tanpa autofocus/auto-pay; periksa unknown tidak membuka pembayaran pengganti; buka 20-item receipt, cetak/batalkan cetak dan pastikan tidak ada sale baru. Recovery setelah keluar route/reload/crash dan kontrak harga tetap release blocker. Tabel history compact dan audit focus menyeluruh masih pekerjaan terpisah.

### Pembaruan berikutnya: FE-R1 shell/header responsive

Baseline berikutnya adalah `b31225b` di develop: i18n dan sebagian navigasi mobile telah digabung sebelum patch ini. FE-R1 dikembangkan terpisah di branch lokal `fix/52-responsive-shell-header`; tidak mengubah logika/controller pembayaran.

- Header memakai tinggi alami dan dua baris di bawah 480 CSS px. Tenant asli dapat wrap (tanpa fallback toko contoh), pemilih bahasa berupa native select berlabel, logout icon memiliki accessible name. Tombol menu/logout dan select setinggi 48px. Identitas role tersedia di sidebar/drawer, bukan memadati header.
- Sidebar permanen dan POS split berbagi breakpoint `xl` 1280px. Di bawahnya, navigasi utama berupa native modal dialog dengan label, tombol tutup, Escape/backdrop dan focus restoration bawaan browser; drawer tertutup tidak menyisakan link offscreen yang masih aktif. Resize ke desktop menutup drawer dan mengarahkan fokus ke navigasi desktop. Interaksi perangkat ini belum diverifikasi manual.
- Menggunakan logical start/end/border dan teks ID/EN/AR untuk label shell baru. `pos.catalog.title` yang sebelumnya tidak terdaftar sekarang tersedia pada ketiga kamus.
- Katalog/cart memakai tinggi konten alami dan scroll halaman, bukan tinggi tetap berdasarkan `100vh`. Grid katalog menggunakan lebar minimum kartu 14rem (dibatasi 100% container), sehingga jumlah kolom mengikuti ruang setelah sidebar/cart. Mode katalog/cart compact yang sudah ada dipertahankan sampai 1280px. Cart shortcut memperhitungkan safe-area bawah dan mengizinkan isi membungkus.
- Badge koneksi statis di header/POS dihapus; belum ada klaim bahwa API dapat dijangkau atau transaksi offline tersedia. Tidak menambah ping endpoint, dependency, layanan hosting, atau desain visual baru.

Verifikasi FE-R1: 31 tes existing termasuk i18n parity lulus, lint/build FE exit 0, backend build/vet/race-short exit 0 (cached), dan diff whitespace lulus. CSS hasil build diperiksa untuk auto-fill grid, breakpoint 1280, `100dvh`, dan safe-area. Ini bukan pengukuran rendered layout atau pengujian focus browser. Tidak ada pengujian browser otomatis atau deployment.

Acceptance manual masih terbuka: (1) 320/390px—header, tenant panjang, bahasa dan logout; (2) 768/1024—drawer dan katalog tidak terjepit; (3) 1280/1440—sidebar dan split tampil bersama; (4) buka drawer, Tab/Escape/tutup/backdrop, resize saat terbuka dan ganti AR; (5) cart 20 item dan keyboard layar tidak tertutup shortcut. Gunakan lingkungan/data uji, bukan uang nyata. FE-R2/R3 tetap mencakup kenyamanan kartu/stepper, focus pergantian katalog-cart, dialog pembayaran, receipt dan review perangkat. Jangan menandai seluruh responsive acceptance selesai.

### Catatan FE-S1 sebelumnya

### Pembaruan FE-R2: kontrol touch dan navigasi compact

Patch lanjutan lokal di atas FE-R1, 2026-09-05; belum commit/publikasi. Kartu produk sekarang berupa article dengan satu native tombol Tambah: tidak ada tombol bersarang atau klik ganda dari event bubbling kartu. Nama panjang tidak di-clamp, SKU dapat wrap, harga boleh turun baris, dan ukuran teks informasi utama ditingkatkan. Tombol tambah, plus/minus/hapus serta search/kategori memakai target 48px. Minus pada jumlah satu tidak menghapus barang, baik pada handler UI maupun updater cart produksi; tombol Hapus tetap eksplisit.

Navigasi katalog/cart mempunyai `aria-pressed`, region berlabel dan pemindahan fokus setelah panel tampil. Posisi scroll katalog disimpan; saat kembali, fokus diarahkan ke tombol yang terlihat di area katalog tanpa membuka keyboard input. Jika tidak ada kontrol yang terlihat, fallback ke heading yang digulir ke area terlihat. Resize yang menyembunyikan kontrol aktif diarahkan ke heading panel yang tetap ditampilkan; tidak mengubah cart/tender/key. Mekanisme ini belum diuji pada browser/perangkat.

Shortcut fixed bawah FE-R1 diganti navigasi compact + total di atas workspace. Navigasi sticky hanya ketika tinggi layout viewport minimal 600px; pada viewport pendek mengikuti document flow. Ini mengurangi overlay bawah, tetapi variasi visual viewport/keyboard mobile tetap memerlukan review manual. Browser Back keluar route, restore setelah reload, dan pemulihan focus setelah menghapus baris cart belum dituntaskan oleh paket ini. Tidak mengubah controller pembayaran atau memberi janji transaksi durable.

Teks baru dan nama aksesibel produk disediakan dalam ID/EN/AR dengan skema/paritas yang sama. Tiga tes baru menjalankan kode komponen/handler/updater produksi dengan adapter test React/i18n/transport UI seperlunya—bukan browser, tes visual, atau simulasi DOM. Tes memeriksa satu aksi tambah, guard disabled, minus satu, batas stok UI, dan penghapusan eksplisit. Seluruh tes lama dipertahankan.

Acceptance manual berikutnya: nama/SKU panjang dan angka besar pada 320/390px; tekan tambah sekali; minus pada qty 1 lalu Hapus; masuk cart dari katalog panjang dan kembali; rotate pada panel aktif; gunakan Tab dan keyboard layar; ganti AR; periksa perubahan header/nav pada tinggi pendek. Gunakan tenant/data pengujian. FE-R3 (dialog pembayaran/receipt, focus lifecycle, kembalian utama) tetap menjadi paket berikutnya; tidak ada klaim seluruh R-01–R-10 sudah lulus.

Pembaruan 2026-09-05 setelah review awal: patch keamanan checkout telah dibuat, belum di-commit. Temuan awal di bawah dipertahankan sebagai baseline; status perubahan terbaru pada bagian ini. **FE-S1 keseluruhan belum selesai dan belum menjadi izin transaksi uang nyata.**

- F-01 diperbaiki untuk satu halaman POS yang masih aktif: API client mempertahankan code/status/trace ID; controller produksi mengklasifikasikan network/5xx, auth, rate limit dan konflik idempotency sebagai terkunci, bukan penolakan bisnis biasa. Hanya kode penolakan pre-commit yang dikenali mengizinkan kembali ke cart dan review baru. Tidak ada retry otomatis.
- Controller menyimpan key serta snapshot payload immutable dalam memori, dan mengunci submit secara sinkron sebelum transport dipanggil. Close/review baru/edit tender/submit lanjutan tidak dapat mengubah transaksi pending atau unknown. Workspace di belakang dialog dibuat inert. Unknown menyembunyikan form bayar dan kembalian preview, serta menampilkan referensi command untuk rekonsiliasi oleh penanggung jawab.
- F-02: `onClose` receipt dan tombol transaksi baru memakai satu aksi `finishCompleted(clearCart)`. Aksi hanya berlaku pada confirmed, mengosongkan cart berbayar sekali, mempertahankan receipt server dalam memori sampai review berikutnya, dan tidak mengirim request pembayaran. Escape/backdrop tidak lagi hanya menghapus state checkout sambil membiarkan cart berbayar.
- Bagian parser F-03: tender memakai `parseIDR`, menjaga input mentah saat invalid, menolak negatif/pecahan/grouping salah serta nominal di luar batas. Controller juga memvalidasi integer/batas sebelum mengirim. Refetch/repricing cart dan kontrak harga server **belum** diselesaikan.
- Modul baru [checkout controller](../../frontend/src/features/pos/checkout-controller.ts) dipakai langsung oleh hook React dan [tes checkout](../../frontend/src/features/pos/checkout.test.mjs). Tes adapter memakai kode produksi yang ditranspilasi dengan TypeScript existing; transport/logger diganti hanya di test scope, tanpa mock endpoint produksi atau dependency baru. Tes POS lama tidak dihapus.

Verifikasi patch: **31 unit test lulus, 0 skip**, frontend lint/build exit 0, backend build/vet/race-short exit 0 (paket Go cached), dan `git diff --check` lulus. Sempat ada lint failure pada nama variabel test `module`; diperbaiki dengan penamaan lokal, tanpa menonaktifkan aturan. Warning module type dan deprecation `next lint` existing tetap ada.

Batas bukti: tes memverifikasi controller/adapter/parser, bukan klik DOM, focus, tampilan, printer, atau transaksi backend nyata. Tidak menjalankan browser otomatis. Recovery setelah unmount/navigasi/Browser Back keluar route, reload, tab ditutup, logout, crash dan multi-tab **belum durable**; memori controller dapat hilang. Instruksi UI bukan pengganti storage dan idempotency recovery. Jangan menutup G-02/G-05 atau mengklaim exactly-once dari patch ini. Copy keselamatan baru masih bahasa Indonesia; review EN/AR mengikuti paket i18n.

Langkah berikut: pisahkan patch dari perubahan i18n pada review Git yang disetujui, lanjutkan rekonsiliasi harga/recovery sebagai subtask FE-S1, dan kerjakan FE-R1 shell/header responsive sebagai paket UI terpisah. Tidak ada endpoint atau kontrak wire API yang diubah oleh patch FE ini.

## Kesimpulan dan posisi proyek

FE bukan lagi scaffold: Next.js 15.5/React 19, tokens CSS, shell, login/BFF, katalog, cart, tender, receipt dan history sudah memiliki kode. Keberadaan kode dan build yang lulus belum membuktikan alur bisnis aman atau pengalaman mobile layak.

Desain sebelumnya sudah mengatur HP, tetapi implementasi menyimpang pada pemisahan katalog/cart, tinggi konten, ukuran kontrol, focus dialog dan penanganan hasil transaksi. Masalah utamanya adalah penerjemahan spesifikasi ke komponen serta kurangnya tes yang menggunakan logika produksi, bukan ketiadaan wireframe baru.

Pertahankan arah visual yang sudah diimplementasikan; jangan memulai redesign seluruh aplikasi. Prioritas: keselamatan transaksi, shell responsive, alur POS compact, lalu konsistensi komponen. Offline cash dan ekspansi modul bukan langkah pertama.

## Temuan keselamatan transaksi — P0, sebelum uang nyata

### F-01: hasil jaringan ambigu masuk jalur penolakan biasa

Sumber: [API client](../../frontend/src/lib/api.ts) `apiClient.post`, [checkout hook](../../frontend/src/features/pos/hooks/use-checkout.ts) `submitCheckout/startReview/closeReview`, [tender](../../frontend/src/features/pos/components/tender-modal.tsx).

Client menangkap `NetworkError` dan mengembalikan `success: false`. Hook memetakan semua hasil tersebut menjadi `rejected`; jalur `catch` untuk `unknown_error` tidak menangani kegagalan jaringan normal yang sudah ditangkap client. Informasi HTTP status/trace ID juga tidak diteruskan dalam envelope client.

Dialog mengizinkan tutup dan submit ketika tidak `isSubmitting`, termasuk unknown. Membuka review lagi menghasilkan key baru. Input/payload juga tidak dibekukan untuk retry. Akibat yang berisiko: request pertama mungkin sudah commit, tetapi user dapat memulai command baru untuk cart yang sama. Ini jalur risiko dari sumber, bukan klaim bahwa double-charge telah direproduksi.

Perbaikan: klasifikasi transport/5xx, auth, validasi bisnis dan idempotency conflict berdasarkan kontrak; state machine eksplisit; command key/body immutable; pengunci submit sinkron; unknown tidak boleh berubah menjadi penjualan baru lewat close/Back/reset. Recovery lintas reload dan tenant harus ditetapkan bersama G-02/G-05. Jangan menambah endpoint status rekaan atau menyebut penyimpanan berhasil sebelum benar-benar tersimpan.

### F-02: receipt dapat ditutup tanpa mengakhiri cart yang sudah dibayar

Sumber: [halaman POS](../../frontend/src/app/(dashboard)/pos/page.tsx), [receipt](../../frontend/src/features/pos/components/receipt-modal.tsx), [Modal](../../frontend/src/components/ui/modal.tsx).

`ReceiptModal.onClose` memanggil `resetCheckout`, sementara cart hanya dikosongkan oleh `handleNewTransaction`. Menyembunyikan tombol X tidak mematikan Escape/backdrop pada Modal. Jalur tersebut menghapus receipt/state pembayaran tetapi membiarkan cart lama bisa dibayar lagi.

Perbaikan: lifecycle confirmed yang konsisten untuk semua jalur dismissal, simpan akses ke receipt server, dan mulai cart baru hanya lewat transisi yang jelas. Menutup/cetak ulang tidak boleh memicu checkout baru.

### F-03: review harga dan parser tender belum mengikuti kontrak desain

Sumber: `handleOpenTender` pada halaman POS, [cart hook](../../frontend/src/features/pos/hooks/use-cart.ts), tender modal dan [money utility](../../frontend/src/lib/money.ts).

Refetch katalog tidak ditunggu dan tidak merekonsiliasi snapshot produk dalam cart. Karena itu komentar mitigasi G-01 bukan bukti harga review sudah mutakhir; refetch saja pun tidak menghilangkan race harga server. Tender membuang semua non-digit, sehingga masukan negatif/desimal dapat berubah arti, serta tidak memakai parser batas nominal bersama.

Perbaikan: parser dan batas uang tunggal, rekonsiliasi harga/stok sebelum review, minta konfirmasi perubahan yang relevan. Tetap selesaikan kontrak quote/expected-price atau kebijakan selisih server; jangan menutup G-01 hanya dengan GET ulang.

## Temuan responsive dan aksesibilitas — P1

### F-04: ukuran viewport dipakai tanpa memperhitungkan ruang kerja aktual

Sumber: halaman POS, [dashboard layout](../../frontend/src/app/(dashboard)/layout.tsx), [sidebar](../../frontend/src/components/layout/sidebar.tsx), [catalog](../../frontend/src/features/pos/components/catalog-grid.tsx).

Workspace memakai `h-[calc(100vh-190px)] min-h-[560px]`, sementara katalog dan cart sama-sama `h-full` saat `flex-col`. Katalog/cart juga punya scroll sendiri. Ini berisiko menghasilkan panel terjepit, halaman panjang dan scroll bersarang pada HP; berbeda dari satu document scroll dalam spec.

Sidebar 240px mulai tampil permanen pada 768px, tetapi split POS dimulai pada 1024px dengan cart 380px. Pada viewport 1024px, perkiraan ruang katalog tinggal 256px setelah sidebar, padding main 64px dan gap 20px, namun grid meminta tiga kolom. Ini perhitungan anggaran layout, bukan pengukuran rendered DOM.

Perbaikan: tinggi otomatis pada compact; satu panel tugas aktif di HP; sidebar drawer sampai workspace cukup luas; split dan jumlah kolom mengikuti **lebar konten tersisa**, bukan nama perangkat. Gunakan satu definisi breakpoint/shell yang konsisten; jangan memperbaiki dengan `overflow-x-hidden` global.

### F-05: header dan kartu tidak punya anggaran ruang compact

Sumber: [header](../../frontend/src/components/layout/header.tsx), [product card](../../frontend/src/features/pos/components/product-card.tsx), [cart panel](../../frontend/src/features/pos/components/cart-panel.tsx).

Identitas lengkap, tenant, tiga pilihan bahasa dan logout tetap berbagi satu baris header pada HP. Grid dimulai dengan dua kolom bahkan pada 320px; nama cart di-truncate, nama produk di-clamp dan SKU tidak diberi pemecahan kata. Harga dan tombol tambah juga berbagi baris sempit.

Perbaikan: tenant nyata yang ringkas tetapi dapat dibuka lengkap, language selector compact melalui kontrol tunggal, identitas sekunder dipindah ke menu. Grid mulai satu kolom; tambah kolom jika ukuran minimum kartu masih layak. Nama/SKU wrap atau punya detail touch/keyboard; angka uang tidak di-ellipsis. Logout icon-only tetap memiliki accessible name.

### F-06: target sentuh dan dialog belum memenuhi baseline proyek

Stepper cart memakai 24×24px, tombol hapus sekitar 22px, tambah produk 32px; category pills 36px. Search memakai teks 12–14px, Input shared 14px. Target POS proyek 48px dan input 16px belum diterapkan merata.

Modal memakai center alignment dengan konten yang dapat lebih tinggi dari viewport; belum ada pengelolaan focus masuk/terkurung/kembali atau hubungan judul dengan `aria-labelledby`. Sidebar tertutup hanya digeser keluar layar, tidak menonaktifkan focus link. Tender autofocus juga bertentangan dengan spec touch. Utility `h-13`, `shadow-xs`, `ring-3` digunakan sementara konfigurasi Tailwind 3 tidak mendefinisikan ekstensi tersebut; audit CSS hasil build diperlukan, jangan menganggap class yang tertulis pasti bekerja.

Perbaikan: kontrak kontrol touch bersama; dialog compact yang dapat digulir pada tinggi pendek, fokus dan dismissal eksplisit, background inert ketika modal aktif, serta drawer yang tidak menyisakan link tersembunyi dalam tab order. Jangan langsung menambah library kedua; evaluasi primitive existing atau satu implementasi accessible yang dipelihara bersama. Hormati reduced motion.

### F-07: informasi utama receipt dan perubahan bahasa perlu reflow

Kembalian berada setelah seluruh daftar item dalam scroll receipt. Letakkan status confirmed, nomor transaksi dan kembalian lebih awal; detail item boleh disclosure, aksi cetak/transaksi baru tetap terjangkau. Riwayat memiliki scroll tabel lokal, tetapi pola blok berlabel untuk HP belum diterapkan.

Perubahan i18n mengaktifkan `dir=rtl`, sedangkan banyak posisi masih `left/right`, `pl/pr` dan `text-left`. Review Arab memerlukan logical properties serta isolasi arah SKU, nomor transaksi dan uang; penggantian kamus saja tidak membuktikan RTL layak. Pertahankan state cart/tender saat mengganti bahasa.

## Drift design system dan bukti pengujian — P2

- Tokens CSS sudah ada, tetapi banyak primitive/header memakai hex langsung; language selector merujuk beberapa `--color-bg-*` yang tidak didefinisikan dalam tokens CSS yang ditinjau. Rapikan lewat semantic alias yang disepakati, bukan palette baru setiap feature.
- `globals.css` masih memiliki auto dark background sementara baseline tokens berorientasi light. Tentukan satu baseline light yang konsisten sebelum menjanjikan dark mode.
- Header memiliki fallback tenant contoh dan label koneksi statis. Ganti dengan identitas/status yang benar-benar diketahui; keterjangkauan API berbeda dari indikator jaringan perangkat. Badge halal harus mengikuti batas bukti kontrak katalog, bukan menjanjikan sertifikasi aktif.
- `pos.test.mjs` mendefinisikan ulang helper cart/change, bukan mengimpor hook/logika produksi. Tes tersebut dapat lulus ketika implementasi produksi menyimpang. Ekstrak logika murni yang benar-benar dipakai hook lalu uji modul itu; pertahankan seluruh skenario tes existing.
- Catatan lama “FE scaffold/belum diimplementasikan” tidak lagi menjadi status implementasi terkini. SVG/prototype sintetis tetap referensi desain, bukan bukti QA aplikasi Next.js.

## Backlog pelaksanaan bertahap

ID berikut adalah checklist kerja lokal/repository, **bukan nomor issue GitHub atau issue draft**. Estimasi merupakan jam fokus implementasi + review + tes terarah oleh satu pengembang; bukan SLA atau prediksi token. Urutan default serial.

- [ ] **FE-R0 — Isolasi baseline dan sinkronisasi status** (1–2 jam): selesaikan/review perubahan i18n secara terpisah, rekam baseline, lalu gunakan branch per task dari develop. Jangan memindahkan atau menggabungkan perubahan lokal secara diam-diam. Output: batas diff dan acceptance setiap paket jelas.
- [ ] **FE-S1 — Keselamatan checkout** (12–24 jam untuk FE; dependency backend terpisah): tangani F-01–F-03 dan tes logika produksi untuk network error, 5xx, rejection, double invocation, immutable retry, dismissal receipt dan nominal invalid. Exit: jalur ambiguous tidak membuka command baru, confirmed tidak kembali menjadi cart belum dibayar. G-01/G-02/G-05 yang belum terbukti tetap release blocker.
- [ ] **FE-R1 — Shell/header dan ruang kerja responsive** (4–8 jam): tangani F-04/F-05 pada shell; sidebar compact, header wrap, lebar konten minimum, hilangkan tinggi desktop pada compact. Exit: pengguna dapat mengakses navigasi dan POS tanpa memaksa kolom sempit; state domain tidak diubah.
- [ ] **FE-R2 — POS compact dan kontrol touch** (6–10 jam): katalog ↔ cart satu state, cart summary terjangkau, grid adaptif, nama panjang, angka besar, ukuran kontrol dan label. Exit: HP tidak perlu melewati seluruh katalog untuk mereview cart; resize/bahasa tidak mereset input. Jangan render dua checkout aktif.
- [ ] **FE-R3 — Dialog, receipt dan history compact** (6–12 jam): F-06/F-07; focus lifecycle, tinggi pendek, kembalian utama, header/aksi wrap, history berlabel. Exit: konten panjang dan keyboard dapat dijangkau; F-02 tidak kambuh.
- [ ] **FE-R4 — Konsistensi token, RTL dan acceptance manual** (4–8 jam): status jujur, token tidak hilang, logical properties, teks lintas bahasa, reduced motion, review perangkat oleh maintainer. Exit: hasil nyata dicatat sebagai pass/fail/not tested per skenario, bukan “responsive selesai” dari build saja.

Total paket di atas **33–64 jam fokus**, belum termasuk backend recovery/quote, penambahan perangkat uji atau revisi besar di luar scope. Pada 10 jam/minggu sekitar 4–7 minggu setelah pembulatan; pada 20 jam/minggu sekitar 2–4 minggu. Kalibrasi ulang sesudah FE-R1; tidak mengulang estimasi seluruh Phase 3 sebagai pekerjaan yang semuanya belum dimulai.

Sesudah paket tersebut: penerimaan POS online terhadap backend nyata → offline **draft** dengan ownership/restore → operasi inventory/procurement → finance/dashboard berdasarkan kontrak. Offline cash tetap menunggu keselamatan harga, durability, idempotency dan rekonsiliasi. Halaman yang belum punya implementasi tidak mendapat tombol navigasi aktif seolah siap digunakan.

## Verifikasi tanpa browser otomatis

AI melakukan review diff/sumber, unit test logika produksi, lint, type/build dan pemeriksaan backend CLI. Tidak menjalankan Playwright, screenshot loop, navigasi browser otomatis atau memasang layanan visual testing untuk pekerjaan ini. Pengujian manual perangkat tetap diperlukan sebelum klaim mobile-ready.

Checklist maintainer, bertahap setelah patch terkait:

1. Pada 320/390 CSS px: header, search, satu item dan cart 20 item; tidak ada page overflow, nama/uang utuh, aksi terjangkau. Gunakan data pengujian non-produksi.
2. Pada 768/1024/1440: ruang katalog cukup, tidak ada tiga kolom terjepit oleh sidebar/cart; aksi dan total sama dengan HP.
3. Putar HP ke landscape; buka keyboard tender dan error; field/aksi dapat digulir, tidak ada submit hanya karena Done atau rotate.
4. Ubah ID/EN/AR, nama tenant/produk panjang, navigasi keyboard dan zoom; pastikan teks, RTL, focus dan cart tetap benar.
5. Pada lingkungan uji saja: validasi unknown/confirmed/dismissal/print cancel tanpa mengulang penjualan. Jangan menguji transaksi berisiko di tenant berisi uang nyata.

Catat viewport, OS/browser, state, expected/actual dan satu screenshot masalah bila diperlukan. Bukti manual belum dikumpulkan dalam review ini. Tidak perlu mengirim seluruh layar/riwayat transaksi atau data sensitif.

Hasil CLI pada snapshot review ini:

- `frontend: npm run test`: **13 lulus**, tidak ada skip; keterbatasan tes duplikasi POS dijelaskan di atas.
- `frontend: npm run lint && npm run build`: **exit 0**, termasuk pemeriksaan tipe. Ada pemberitahuan deprecation `next lint`; unit test mengeluarkan peringatan module type. Bukan bukti layout/focus/browser lulus.
- `backend: go build ./... && go vet ./... && go test -race -short ./...`: **exit 0**, hasil paket Go cached. Ini bukan full E2E atau bukti penutupan recovery gate.

## Handoff dan pengelolaan perubahan

Artefak kanonik tetap source React/CSS, Markdown, tokens dan SVG lokal. Tidak membutuhkan akun Figma/Penpot, plugin berbayar, AI runtime atau hosting tertentu untuk perbaikan ini. Tidak perlu menggambar ulang seluruh mockup sebelum memperbaiki penyimpangan yang sudah jelas; perbarui spec hanya ketika perilaku yang disepakati berubah.

Pisahkan review/docs, i18n, keselamatan pembayaran dan responsive ke diff kecil. Issue/commit/PR/push/merge mengikuti persetujuan eksplisit repository; dokumen ini tidak mengotorisasi remote mutation. Catatan model/quota dan handoff sesi disimpan terpisah di `docs/local/` yang diabaikan Git.
