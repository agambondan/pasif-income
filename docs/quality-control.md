# Quality Control

Walaupun sistemnya otomatis, output tetap harus dijaga kualitasnya.

Checklist dasar:

- Skrip tidak berulang.
- Voiceover enak didengar.
- Visual sesuai narasi.
- Caption terbaca jelas.
- Branding konsisten.

Kalau quality control lemah, konten akan terlihat seperti spam.

## Current Implementation

Quality control sekarang berjalan sebagai gate sebelum upload:

1. Generator merender video.
2. QC service memeriksa title, script, scenes, voiceover, dan file video.
3. Jika tersedia, `ffprobe` dipakai untuk cek durasi dan orientasi video.
4. Jika kredensial Gemini tersedia, reviewer AI menilai ulang struktur dan flow konten.
5. Kalau QC gagal dan auto-regenerate aktif, generator mencoba satu kali revisi prompt lalu render ulang.

Hasil QC ditulis ke log runtime. Bila gate gagal pada semua percobaan, upload dibatalkan.
