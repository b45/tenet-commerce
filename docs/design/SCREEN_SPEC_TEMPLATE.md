# Template Spesifikasi Layar / UI Pattern

Gunakan bersama [Pedoman FE](../FRONTEND_GUIDELINES.md). Salin hanya bagian relevan; untuk perubahan kecil cukup catat delta terhadap spesifikasi sebelumnya. Bagian kosong di template bukan endpoint atau fitur yang terdaftar.

## Identitas dan tujuan

- Screen/UX ID dan nama:
- Status: proposed / reviewed / ready / implemented / verified.
- Revision dan tanggal:
- Role, permission dan tujuan pengguna:
- Kriteria keberhasilan tugas:
- Pattern existing yang dipakai:
- Source design dan token revision:
- Keputusan terbuka/dependency:

## Data dan kontrak

| Field/informasi | Sumber endpoint/DTO atau state lokal | Format/unit | Validasi/permission |
|---|---|---|---|

Catat data unavailable/null versus nol. Bedakan field tampilan dari payload. Jangan menambah endpoint, metric atau form input yang tidak didukung tanpa dependency eksplisit.

## Struktur dan perilaku

- Urutan region dan informasi utama:
- Aksi primer/sekunder/destructive, masing-masing dengan hasilnya:
- Layout desktop/tablet/compact dan perilaku pada zoom:
- Overflow untuk nama panjang, tabel, dan jumlah besar:
- Keyboard/focus order, shortcut, accessible labels:
- Konvensi money/date/timezone dan teks pesan:
- Receipt/print behavior jika relevan:

## Matriks state

| Trigger/state | Tampilan dan pesan | Aksi tersedia | Recovery/data dipertahankan | Bukti test |
|---|---|---|---|---|

Pertimbangkan default, loading, empty, invalid, pending, success, stale, forbidden, session expired, server error, offline dan outcome unknown. Tandai N/A dengan alasan; jangan membuat varian yang tidak relevan hanya untuk mengisi tabel.

## Reuse dan keputusan visual

- Komponen/variants existing:
- Token foreground/background yang dipakai:
- Pasangan kontras yang diuji:
- Kebutuhan komponen/variant baru dan alasannya:
- Penyimpangan dari default pedoman serta bukti kebutuhan:

## Acceptance examples

```text
Given: kondisi awal, data dan permission
When: tindakan pengguna atau perubahan koneksi/session
Then: hasil terlihat, efek backend dan state lokal yang diharapkan
Evidence: test/rekaman/catatan inspeksi yang benar-benar tersedia
```

Sertakan happy path dan kegagalan bermakna. Pada mutation, jelaskan key/retry/reconciliation di spesifikasi teknis tanpa memaksa istilah itu menjadi label kasir.

## Handoff dan verifikasi

- Implemented files dan API dependency yang terselesaikan:
- Perintah tes serta hasil aktual:
- Keyboard/screen reader, viewport/zoom, perangkat/printer yang diperiksa:
- Native source/export revision dan hasil restore:
- Kekurangan yang masih terbuka:

Jangan mengubah status menjadi verified hanya karena mockup tampak selesai. Catatan personal/model/quota berada di docs/local/, bukan spesifikasi publik ini.
