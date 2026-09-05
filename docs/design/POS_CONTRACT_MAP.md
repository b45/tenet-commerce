# POS — peta kontrak untuk desain pertama

Revision `pos-lowfi-0.1`, 2026-09-05. Inspeksi source, bukan verifikasi API berjalan. Base path `/api/v1`. Tidak menambah/mengubah endpoint.

Sumber: [router](../../backend/cmd/api/router.go), [handler POS](../../backend/internal/pos/handler.go), [DTO](../../backend/internal/pos/models.go), [service](../../backend/internal/pos/service.go), [repository](../../backend/internal/pos/repository.go), [roles](../../backend/pkg/auth/jwt.go), [envelope](../../backend/pkg/response/response.go), [handler auth](../../backend/internal/auth/handler.go).

## Scope dan permission

Target pertama: CASHIER; MANAGER/SUPER_ADMIN juga memiliki permission terkait. `inventory:read` untuk katalog, `pos:checkout` untuk penjualan, `pos:read` untuk receipt/history. COMPLIANCE_OFFICER dan FINANCIAL_ADMIN tidak memiliki `pos:checkout`; jangan membuka pembayaran hanya karena dapat membaca katalog. Server tetap otoritas permission.

Di luar paket: diskon manual, customer/notes, void/refund, QRIS, kartu simulasi, shift closing, offline cash dan history screen lengkap. Sebagian tersedia di API tetapi sengaja belum didesain. Role CASHIER memang memiliki `pos:void` saat ini; jangan mengarang aturan manager-only tanpa keputusan backend.

| Operasi | Kontrak nyata | Pemakaian / batas desain |
|---|---|---|
| GET `/auth/me` | Handler mengisi `id`, `tenant_slug`, `role`, `permissions`; `full_name`/`email` pada DTO tidak diisi di handler ini | Nama header boleh dari profil login/refresh; bila tidak tersedia tampilkan ID kasir, jangan mengarang nama. Slug bukan nama/alamat merchant |
| GET `/pos/products` | `data: Product[]`, `meta.total` panjang hasil; active products, tidak menerima search/pagination | Search nama/SKU/barcode dan filter kategori dilakukan pada snapshot yang telah dimuat; bukan pencarian server |
| GET `/pos/categories` | Kategori dengan `id`, `name`, `code`, `parent_id`, `product_count` | Opsional; baseline boleh menurunkan pilihan kategori dari katalog yang dimuat, tanpa hit API tambahan |
| POST `/pos/checkout` | `pos:checkout`, `Idempotency-Key`; berhasil HTTP 201, `data: CheckoutResponse` | Satu command immutable per percobaan bisnis; jangan kirim ulang dengan key baru ketika hasil belum jelas |
| GET `/pos/orders/:id` | `pos:read`; menerima UUID atau nomor transaksi; `data.transaction` dan `data.items` | Receipt/detail saat identitas server sudah diketahui. Bukan lookup idempotency key |
| GET `/pos/orders` | `limit`, `offset`, `start_date`, `end_date`, `status`, `payment_method`, `search`; meta total/limit/offset | Search mencocokkan nomor transaksi/customer/notes; tidak mendukung pencarian key. Review kandidat manual tidak membuktikan kecocokan command |

Envelope umum `{success, data?, error?: {code,message,details?}, meta?}`. Normalisasi katalog kosong `data: null` menjadi daftar kosong setelah sukses; slice Go dapat nil. Field opsional kosong tidak dianggap nilai nol. Jangan tampilkan SQL/raw `error.details` ke kasir.

## Pemetaan field

| Informasi UI | Sumber | Aturan |
|---|---|---|
| Nama/SKU/barcode | `Product.name/sku/barcode` | Barcode/SKU string, pertahankan leading zero; exact match untuk scanner, hasil ambigu tidak auto-add |
| Kategori | `category_id/category_name` | Missing → “Tanpa kategori” |
| Harga/stok | `unit_price`, `stock_quantity` | Snapshot, bukan reservasi; jangan tampilkan `cost_price` dalam POS |
| Tag halal | `is_halal_certified`, `compliance_tags` | Flag diturunkan dari tag HALAL_MUI; bukan pemeriksaan sertifikat aktif. Tidak ditampilkan dalam tiga frame awal |
| Jumlah/baris/subtotal estimasi | State cart + snapshot produk | Integer positif, merge SKU yang sama; hitung integer rupiah dengan batas aman |
| Pembayaran | `payment_method: CASH`, `cash_tendered` | UI meminta tender eksplisit ≥ estimasi; backend juga memvalidasi terhadap harga aktual |
| Bukti hasil | `transaction_id`, `transaction_number`, `status`, `created_at` | Sukses hanya dengan payload valid dan status `COMPLETED`; status lokal unknown bukan enum transaksi server |
| Angka receipt | `items`, `subtotal_amount`, `tax_amount`, `discount_amount`, `total_amount`, `cash_tendered`, `change_amount` | Semua dari receipt server; jangan mencetak ulang dari harga katalog saat ini |

Payload contoh sintetis untuk cart di wireframe; ini dokumentasi, bukan endpoint dummy:

```json
{
  "items": [{"sku": "SUSU-001", "quantity": 2}, {"sku": "ROTI-001", "quantity": 1}],
  "payment_method": "CASH",
  "discount_amount": 0,
  "cash_tendered": 50000
}
```

Header key dibuat client saat command disiapkan, bukan ketika memasuki halaman. Simpan original payload/key/tenant/user sebelum submit; detail storage/session mengikuti P3-02. Jika penyimpanan gagal, jangan mulai pembayaran. Token tidak disimpan bersama command.

## Error dan tindakan aman

| HTTP / code | Interpretasi dan UI |
|---|---|
| 400 `VALIDATION_ERROR` | Periksa input; pertahankan cart. Jangan tampilkan binding internals |
| 400 `INSUFFICIENT_CASH_TENDERED` | Uang belum cukup untuk total server; refresh/review harga, minta koreksi tender setelah penolakan pasti |
| 400 `INVALID_CASH_TENDERED` | Payload/metode tidak sesuai; bug integrasi pada scope CASH, tidak retry loop |
| 404 `PRODUCT_NOT_FOUND` | Produk hilang/nonaktif; refresh dan review item; tidak auto-remove tanpa pemberitahuan |
| 409 `INSUFFICIENT_STOCK` | Penolakan stok; refresh/review cart. Detail item masih pesan teks, bukan struktur SKU yang stabil |
| 409 `CONCURRENT_MUTATION_IN_PROGRESS` | Command sedang diproses; tunggu, jangan menampilkan “stok habis” atau membuat key baru |
| 409 `IDEMPOTENCY_KEY_REUSED_WITH_DIFFERENT_PAYLOAD` | Identitas command bentrok; hentikan dan investigasi original record, bukan regenerate otomatis |
| 401 / 403 | Autentikasi ulang / akses ditolak; pertahankan record terkunci milik tenant/user semula |
| 429 | Tunggu dengan backoff; hormati Retry-After jika ada. Tidak dianggap penolakan stok |
| Timeout, koneksi putus, 5xx, respons sukses tidak valid | Hasil belum diketahui; pertahankan original command. Tidak menagih/membuat checkout baru untuk transaksi ini |

Penolakan bisnis yang pasti dapat menghasilkan command terkoreksi dengan key baru setelah review dan pencatatan hubungan ke percobaan sebelumnya. Respons 4xx dari retry setelah pernah unknown tidak otomatis membuktikan command awal belum committed; tetap perlu rekonsiliasi.

## Dependency sebelum transaksi produksi

| ID | Temuan source dan keputusan yang diperlukan |
|---|---|
| G-01 Harga | Request hanya SKU/jumlah, tidak ada expected price/quote version. Backend mengambil harga saat checkout. Refresh sebelum bayar mengurangi stale, tetapi tidak menutup race. Harga naik masih bisa lolos jika tender cukup. Tentukan kontrak persetujuan harga/quote atau kebijakan rekonsiliasi selisih yang disetujui sebelum release |
| G-02 Unknown outcome | [Middleware](../../backend/pkg/idempotency/middleware.go) menyimpan respons setelah `c.Next()`, sementara service sudah commit. Belum ada route status-by-key; perlu bukti crash recovery, lease expiry, replay setelah cache hilang dan batas retensi. Tombol otomatis “periksa status” belum dapat dijanjikan |
| G-03 Money | [Utility](../../backend/pkg/money/money.go) membulatkan IDR utuh, tetapi DTO float dan beberapa error operasi money di service diabaikan. Tetapkan bounds/rounding; uji overflow dan tender. Form awal menolak pecahan; ini tidak menyelesaikan risiko backend |
| G-04 Catalog | Full active catalog tanpa version/pagination. Validasi skala data dan waktu muat; scan bekerja pada data dimuat. Tidak mengklaim stok realtime atau offline-complete |
| G-05 Session/recovery | BFF/session/CSRF, storage ownership, reload/multi-tab dan retensi belum diimplementasikan. Record pembayaran tidak boleh hilang saat logout atau dikirim di identitas lain |
| G-06 Receipt | Respons punya cashier ID, bukan nama historis kasir atau alamat merchant. Receipt contoh minimal; identitas usaha, timezone konfigurasi dan kebutuhan printer perlu diputuskan. Tidak mengklaim faktur pajak |

Tax service saat ini bernilai nol dan discount di-clamp sampai subtotal. Wireframe bukan kebijakan pajak: scope mengirim diskon nol, tidak menawarkan pengaturan pajak. Receipt menampilkan angka server apa adanya. Tidak ada perubahan API/collection dalam pekerjaan desain ini.
