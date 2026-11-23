# PRD POC Gamifikasi Gacha: WebView vs Native vs Server-Driven UI

Nov 23, 2026 · @Muhammad Faisal Affan

## Latar belakang dan tujuan

POC ini membangun satu fitur gacha poin yang sama dalam tiga pendekatan front-end (WebView, native screen, server-driven UI), lalu memilih satu berdasarkan data terukur, bukan opini.

**Latar belakang.** Mini program gamifikasi (lootbox/gacha dengan hadiah poin) saat ini berjalan di container IAPMiniProgram SDK dan backend Alibaba Cloud. Fitur ini akan dipindah ke container di OCP dan ditanam di super app banking native. Probabilitas hadiah ditampilkan ke user, dan hadiahnya hanya poin tanpa nilai uang.

**Tujuan POC**

- Membuktikan fitur gacha poin berjalan end-to-end dengan backend di OCP.
- Membandingkan tiga pendekatan front-end pada kriteria yang sama (performa, keamanan, kecepatan rilis, biaya).
- Menghasilkan rekomendasi untuk bahan design forum (EAWG/RCB).

**Non-tujuan**

- Bukan migrasi produksi dan tidak memakai data user asli.
- Tidak ada hadiah bernilai uang dan tidak ada undian berbayar. Kalau berubah, review legal dan compliance dilakukan ulang sebelum lanjut.
- Tidak membangun framework SDUI generik, hanya renderer untuk komponen yang dibutuhkan fitur ini.

## Scope fitur referensi

Setiap pendekatan membangun fitur yang sama persis: satu alur gacha poin dari lihat saldo sampai hasil draw, supaya perbandingannya adil.

| # | Fitur | Perilaku yang diharapkan | Prioritas |
| --- | --- | --- | --- |
| F1 | Lihat saldo poin | Menampilkan saldo terkini dari backend | Wajib |
| F2 | Daftar probabilitas | Menampilkan probabilitas hadiah beserta `table_version` yang dipakai | Wajib |
| F3 | Buka kotak (draw) | Satu draw memotong poin dan memberi hasil dari server, dengan idempotency key | Wajib |
| F4 | Animasi buka kotak | Animasi di client, hasil tetap dari server | Wajib |
| F5 | Riwayat draw | Daftar draw terakhir milik user | Opsional |
| F6 | Error dan retry | Saldo kurang, timeout, dan draw duplikat ditangani tanpa memberi hadiah dua kali | Wajib |
| F7 | Capability native | Mengambil session token dari native (bukan dari web storage) | Wajib untuk WebView |

Perilaku bisnis diasumsikan identik dengan mini program saat ini. Spesifikasi detail (jumlah tier hadiah, harga poin per draw) perlu dikonfirmasi dari sistem yang ada sebelum sprint pertama.

## Arsitektur bersama

Satu backend di OCP melayani ketiga pendekatan, jadi logika undian dan saldo poin tidak diduplikasi di client.

&#91;embedded content: arsitektur bersama · 3 klien, 1 backend OCP\]

WebView memuat SPA dari OCP, lalu ketiga pendekatan memanggil API yang sama lewat gateway. Draw, probabilitas, ledger, dan audit hanya ada di backend.

## Pendekatan A: WebView + native bridge

UI berupa web app (SPA) yang di-serve dari OCP dan dibuka di WebView super app. Native hanya menyediakan bridge terbatas.

**Kebutuhan**

- Frontend SPA dengan DOM untuk saldo, daftar, dan tombol. Canvas atau Lottie hanya untuk animasi buka kotak (F4).
- Container frontend di OCP dengan route statis, versi build yang jelas, dan header CSP ketat.
- WebView hanya boleh memuat domain yang di-allowlist, dengan certificate pinning untuk domain itu.
- Bridge JS ke native dengan method whitelist: `getSessionToken` dan seperlunya. Method lain ditolak, dan setiap panggilan divalidasi (origin, skema parameter).
- Session token disimpan native, bukan di localStorage WebView. Web hanya menerima token berumur pendek.
- Mekanisme fallback: native bisa kembali ke jalur lama (mini program) lewat feature flag tanpa rilis ulang app.

**Hal yang diukur khusus**

- Waktu cold start WebView sampai halaman interaktif.
- Latensi bridge (JS ke native dan balik).
- Perilaku saat kembali dari background, kehilangan jaringan, dan versi OS berbeda.

## Pendekatan B: Native screen

Fitur ditulis sebagai layar native di super app dan memanggil API backend OCP langsung lewat API gateway yang sama dengan jalur native lain.

**Kebutuhan**

- Layar native untuk F1 sampai F6, memakai komponen design system app yang sudah ada.
- Animasi buka kotak memakai Lottie atau Rive (atau animasi native), dengan hasil draw tetap dari server.
- Pemanggilan API lewat network layer app yang sudah ada, termasuk certificate pinning dan token management.
- Platform POC: satu platform dulu (diasumsikan Android atau iOS sesuai keputusan tim), platform kedua dinilai dari estimasi usaha tambahan.
- Kalau tim memilih modul Flutter, catat ukuran tambahan app dan biaya integrasi engine.

**Hal yang diukur khusus**

- Waktu dari tap sampai layar interaktif dan frame rate animasi di device kelas menengah.
- Usaha (jam kerja) untuk setiap perubahan UI sederhana, termasuk siklus rilis sampai ke user.
- Ukuran tambahan app.

## Pendekatan C: Server-driven UI

Backend OCP mengirim deskripsi layar (JSON), dan renderer native menggambarnya dengan komponen native. Untuk POC, renderer hanya mendukung komponen yang dibutuhkan fitur referensi.

**Kebutuhan**

- Skema layar bersifat versioned (`schema_version`), dengan katalog komponen minimal: teks, tombol, kartu saldo, daftar probabilitas, dan komponen kotak gacha dengan animasi.
- Renderer native memvalidasi JSON terhadap skema dan menolak komponen atau aksi yang tidak dikenal.
- Aksi (misalnya "draw") dipetakan ke daftar aksi whitelist di native, bukan eksekusi bebas dari JSON.
- Fallback: versi app yang tidak mengenal `schema_version` baru menampilkan layar bawaan atau mengarahkan ke jalur lama.
- Backend menyajikan skema dari service yang sama dengan backend gacha, dengan perubahan lewat pipeline review.

**Hal yang diukur khusus**

- Usaha membangun renderer dan katalog komponen (jam kerja) dan perkiraan biaya merawatnya.
- Seberapa banyak perubahan UI yang bisa dilakukan murni dari backend, dan apa yang tetap butuh rilis app (misalnya komponen animasi baru).
- Perilaku saat versi skema dan versi app tidak cocok.

## Keamanan dan compliance

Keamanan berlaku sama untuk ketiga pendekatan dan menjadi syarat lolos, bukan bahan tawar-menawar.

- **Logika undian hanya di server.** RNG memakai CSPRNG, tabel probabilitas hanya ada di backend, dan client hanya menampilkan hasil.
- **Idempotency.** Setiap draw punya idempotency key, jadi retry tidak memberi hadiah dua kali.
- **Ledger poin.** Saldo dicatat sebagai ledger append-only yang bisa direkonsiliasi, bukan satu angka yang di-update.
- **Audit log.** Setiap draw mencatat user, waktu, hasil, dan `table_version`.
- **Konsistensi probabilitas.** Angka yang ditampilkan ke user berasal dari `table_version` yang sama dengan yang dipakai saat draw.
- **Rate limit dan deteksi automasi** per user dan per device.
- **Data.** POC memakai data sintetis dan tidak menyentuh data user atau saldo produksi.
- **Review.** Pentest WebView (pendekatan A) dan review MASVS dilibatkan sebelum rollout. Ketentuan regulator untuk aplikasi keuangan dan promosi dicek bersama legal dan compliance, karena detail regulasi terbaru belum dikonfirmasi di dokumen ini.
- **Hadiah non-moneter.** Kalau hadiah atau mekanisme bayar-untuk-undi berubah, review legal ulang dilakukan sebelum pekerjaan dilanjutkan.

## Kriteria evaluasi dan keputusan

Pemenang dipilih dari skor tertimbang atas kriteria berikut. Target angka disepakati dengan tim sebelum pengukuran dimulai, bukan sesudahnya, supaya hasil tidak disesuaikan dengan pilihan yang sudah disukai.

| Kriteria | Cara ukur | Target | Bobot (usulan) |
| --- | --- | --- | --- |
| Waktu sampai layar interaktif | Rata-rata dan p95 di minimal 2 device kelas menengah, cold dan warm | Disepakati sebelum POC | 20% |
| Kelancaran animasi (F4) | Frame rate dan jank saat animasi buka kotak | Disepakati sebelum POC | 15% |
| Keamanan | Hasil review dan temuan pentest awal, jumlah attack surface tambahan | Tanpa temuan kritis | 25% |
| Kecepatan perubahan UI | Waktu dari permintaan perubahan sampai live di user, untuk 3 perubahan contoh | Disepakati sebelum POC | 20% |
| Biaya bangun dan rawat | Jam kerja POC, estimasi jam kerja per perubahan dan per platform | Dibandingkan antar pendekatan | 20% |

Bobot adalah usulan awal dan perlu disetujui pemilik produk dan tim keamanan. Pendekatan dengan temuan keamanan kritis yang belum bisa ditutup otomatis gugur, apa pun skornya.

**Hasil POC**

Bagian ini diisi setelah pengukuran selesai: skor per kriteria untuk WebView, native screen, dan SDUI, lalu rekomendasi akhir beserta alasan dan risiko yang tersisa.

## Timeline, risiko, dan open questions

POC berjalan dalam lima fase berurutan dan baru sampai ke keputusan setelah tiga gate terpenuhi. Durasi tiap fase ditetapkan setelah kapasitas tim dan target diketahui.

&#91;embedded content: roadmap POC · 5 fase, 3 gate\]

Gate 2 mencerminkan syarat keamanan di bagian sebelumnya (idempotency dan ledger yang bisa direkonsiliasi), dan gate 3 berarti pendekatan dengan temuan kritis yang belum tertutup gugur.

**Risiko**

| Risiko | Dampak | Mitigasi |
| --- | --- | --- |
| Target dan bobot disesuaikan setelah hasil terlihat | Keputusan sulit dipertanggungjawabkan | Target dan bobot disepakati di Fase 1, sebelum pengukuran |
| Fitur berbeda antar pendekatan | Perbandingan tidak adil | Fitur referensi dibekukan sebelum prototipe dimulai |
| Celah di bridge WebView | Kebocoran token atau aksi tidak sah | Method whitelist, validasi origin, pentest awal |
| Tim mobile menjadi bottleneck untuk B dan C | Jadwal POC molor | Kapasitas mobile dialokasikan di Fase 1 |
| Spesifikasi gacha saat ini belum terdokumentasi | Perilaku POC berbeda dari produksi | Inventarisasi endpoint dan tabel hadiah di Fase 1 |

**Open questions**

- [ ] Platform POC untuk native screen: Android, iOS, atau keduanya?
- [ ] Target angka performa (cold start, frame rate) dan device uji yang dipakai?
- [ ] Siapa pemilik keputusan akhir dan kapan jadwal design forum?
- [ ] Apakah kontrak vendor IAPMiniProgram membatasi pemindahan atau penghentian?
- [ ] Berapa tier hadiah dan harga poin per draw di sistem yang ada?
- [ ] Apakah super app sudah punya framework server-driven UI yang bisa dipakai ulang?

## Pertanyaan untuk tim dan rekomendasi awal

Rekomendasi awal: mulai dengan WebView untuk kondisi saat ini, siapkan jalur ke native untuk layar inti, dan pertimbangkan SDUI hanya kalau fitur serupa bertambah. Rekomendasi ini bersifat hipotesis dan harus dikoreksi oleh jawaban tim di bawah dan hasil pengukuran POC.

**Kenapa begitu**

- **Saat ini:** WebView paling cepat untuk event yang sering berganti, bisa dikerjakan tim web tanpa menunggu rilis app, dan melepas ketergantungan pada SDK. Risikonya ada di bridge dan performa, dan itu yang diukur POC.
- **Nanti, kalau animasi atau performa jadi prioritas:** layar inti (saldo dan hasil draw) dipindah ke native, event tetap di WebView.
- **Nanti, kalau banyak fitur gamifikasi serupa:** SDUI mulai masuk akal karena biaya framework terbagi ke banyak fitur. Untuk satu fitur, biayanya tidak sebanding.

**Pertanyaan untuk tim**

| # | Pertanyaan | Ditujukan ke | Jawaban yang menggeser pilihan |
| --- | --- | --- | --- |
| 1 | Seberapa sering event atau tampilan gacha berganti (mingguan, bulanan, jarang)? | Produk | Sering: WebView atau SDUI. Jarang: Native screen |
| 2 | Apakah animasi buka kotak jadi nilai jual utama atau cukup sederhana? | Produk | Utama: Native screen atau hybrid. Sederhana: WebView |
| 3 | Berapa banyak fitur gamifikasi lain yang direncanakan dalam setahun ke depan? | Produk | Banyak: SDUI layak. Satu atau dua: WebView atau Native screen |
| 4 | Apakah super app sudah punya WebView yang terhardening (allowlist, pinning, bridge)? | Mobile | Sudah: WebView murah. Belum: biaya WebView naik |
| 5 | Apakah super app sudah punya framework server-driven UI? | Mobile | Sudah: SDUI murah. Belum: SDUI mahal |
| 6 | Kapasitas tim mobile untuk POC dan rilis berkala (Android, iOS, atau keduanya)? | Mobile | Terbatas: WebView. Cukup: Native screen |
| 7 | Apakah ada tim web yang bisa memegang frontend, dan stack apa yang dipakai? | Web / Engineering | Ada: WebView. Tidak ada: Native screen |
| 8 | Apakah kebijakan keamanan mengizinkan WebView dengan bridge untuk fitur di dalam app banking? | Security | Tidak: Native screen atau SDUI. Ya, bersyarat: WebView |
| 9 | Apakah ada persyaratan regulator atau audit khusus untuk konten web di app keuangan? | Compliance | Ketat: Native screen atau SDUI |
| 10 | Apakah kontrak vendor IAPMiniProgram membatasi pemindahan atau penghentian, dan kapan berakhir? | Vendor management | Menentukan batas waktu migrasi |
| 11 | Platform OCP apa yang tersedia untuk menyajikan frontend statis dan API (ingress, CDN, WAF)? | Infra | Menentukan biaya WebView dan SDUI |
| 12 | Siapa pemilik keputusan akhir dan kapan jadwal design forum? | Project lead | Menentukan tenggat POC |

**Estimasi timeline (usulan awal)**

Asumsi: satu engineer backend, satu web, satu mobile, dan keamanan paruh waktu. Koreksi bila komposisi tim berbeda. Angka ini belum divalidasi dengan tim.

| Fase | Estimasi | Catatan |
| --- | --- | --- |
| 1. Persiapan | 1 sampai 2 minggu | Menjawab pertanyaan di atas, menetapkan target dan bobot, inventarisasi sistem lama |
| 2. Backend OCP | 3 sampai 4 minggu | Draw, ledger, audit log, dan uji idempotency |
| 3. Prototipe tiga pendekatan | 3 sampai 4 minggu | Bisa mulai paralel dengan fase 2 memakai mock API. SDUI paling lama karena renderer |
| 4. Pengukuran | 1 sampai 2 minggu | Performa di device uji, review keamanan, estimasi usaha |
| 5. Keputusan | 1 minggu | Skor tertimbang dan dokumen untuk design forum |

Total sekitar 9 sampai 13 minggu kalau berurutan, dan sekitar 7 sampai 10 minggu kalau fase 3 berjalan paralel dengan fase 2. Risiko terbesar ada di fase 1 (jawaban tim yang lambat) dan kapasitas mobile.
