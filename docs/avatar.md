# AI Avatar

Di 2026, avatar AI yang konsisten bisa dipakai sebagai identitas visual channel.

Manfaatnya:

- Membuat channel terasa punya wajah atau persona.
- Menjaga konsistensi branding.
- Memudahkan produksi konten dalam skala besar.

Avatar tidak harus realistis, tapi harus stabil dan mudah dikenali.

## Current Implementation

Sekarang `pasif-income` sudah punya branding profile per niche:

- Persona dan watermark ditentukan deterministik dari niche atau override env.
- Avatar AI digenerate sekali lalu di-cache per niche di `branding-assets/<niche>/avatar.png`.
- FFmpeg assembler menambahkan avatar ke video dan menaruh watermark plus intro/outro title card.
- QC agent sekarang ikut memeriksa apakah branding profile dan avatar asset tersedia.

Env yang relevan:

- `BRANDING_ENABLED`
- `BRANDING_ASSET_DIR`
- `BRAND_PERSONA`
- `BRAND_WATERMARK`
- `BRAND_INTRO_TEXT`
- `BRAND_OUTRO_TEXT`
- `BRAND_AVATAR_PROMPT`
- `BRAND_ACCENT_COLOR`
