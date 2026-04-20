# Dynamic Affiliate Insertion

Dokumen ini menjelaskan layer monetization untuk `pasif-income`.

## Tujuan

Fitur ini dipakai untuk menempelkan CTA affiliate yang relevan ke output produksi:

- menambahkan disclosure monetization ke description
- memilih offer yang paling cocok untuk niche/topic
- menyediakan pin comment yang bisa dipakai ulang di platform yang mendukung

## Current Implementation

Sekarang backend sudah membuat affiliate plan per job:

1. Niche dan topic dipakai untuk scoring offer.
2. Offer dipilih dari catalog env atau fallback default.
3. Description video otomatis berisi disclosure, link, dan CTA.
4. Pin comment disimpan di generation job dan tampil di dashboard.

Env yang relevan:

- `AFFILIATE_ENABLED`
- `AFFILIATE_BASE_URL`
- `AFFILIATE_DISCLOSURE`
- `AFFILIATE_CATALOG_JSON`

## Catalog JSON

`AFFILIATE_CATALOG_JSON` menerima array object seperti:

```json
[
  {
    "title": "The Daily Stoic",
    "url": "https://example.com/affiliate/daily-stoic",
    "cta": "Read it before your next scroll session.",
    "disclosure": "Disclosure: this post contains affiliate links.",
    "pin_comment": "If this helped, grab the book here:",
    "tags": ["stoicism", "mindset"]
  }
]
```

## Next Step

Setelah ini, ekstensi paling bernilai adalah:

- per-job tracking untuk offer mana yang menghasilkan klik
- UI editor untuk catalog affiliate
- A/B testing CTA copy
