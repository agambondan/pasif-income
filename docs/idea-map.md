# Idea Map

Dokumen ini menjelaskan layer niche research dan trending topic discovery di `pasif-income`.

## Tujuan

Fitur ini dipakai untuk menjawab pertanyaan:

- topik apa yang lagi naik untuk niche tertentu
- ide video mana yang paling layak dieksekusi sekarang
- query apa yang bisa dipakai langsung sebagai topic produksi

## Current Implementation

Saat ini dashboard bisa melakukan research live dari niche aktif:

1. Mengambil sinyal tren umum dari Google Trends RSS.
2. Mengambil autocomplete suggestion dari Google dan YouTube untuk seed niche.
3. Menggabungkan semua sinyal, memberi skor, lalu menyusun ide video yang bisa dipakai langsung.
4. Menampilkan daftar ide di dashboard dan mengisi field topic ketika ide dipilih.

Endpoint yang tersedia:

- `POST /api/research/ideas`

Request body:

```json
{
  "niche": "stoicism",
  "limit": 5
}
```

Response ringkas:

- `signals`: sinyal tren mentah yang dipakai untuk scoring
- `ideas`: daftar ide yang siap dieksekusi
- `warnings`: warning kalau source eksternal tidak tersedia

## How To Use

- Isi niche di dashboard.
- Klik `DISCOVER IDEAS`.
- Pilih salah satu suggestion untuk mengisi `topic` produksi.

## Next Step

Setelah ini, ekstensi paling bernilai adalah:

- ranking ide berdasarkan performa publikasi sebelumnya
- penyimpanan research history per niche
- A/B testing hook untuk ide yang dipilih
