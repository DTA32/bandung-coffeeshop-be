-- i18n: add Indonesian (`*_indo`) columns alongside the existing English
-- baseline text columns. The original columns remain the English / fallback
-- values; queries prefer the `*_indo` column when lang='id' and fall back to
-- English when it is empty/NULL. Proper-noun `location.name` is NOT translated.

ALTER TABLE location
    ADD COLUMN IF NOT EXISTS description_indo TEXT NOT NULL DEFAULT '';

ALTER TABLE location_image
    ADD COLUMN IF NOT EXISTS description_indo TEXT NOT NULL DEFAULT '';

ALTER TABLE rating_category
    ADD COLUMN IF NOT EXISTS name_indo              TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS short_description_indo TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS long_description_indo  TEXT NOT NULL DEFAULT '';

ALTER TABLE tag
    ADD COLUMN IF NOT EXISTS name_indo        TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS description_indo TEXT NOT NULL DEFAULT '';

ALTER TABLE cafe_review
    ADD COLUMN IF NOT EXISTS content_indo TEXT;

ALTER TABLE cafe_rating
    ADD COLUMN IF NOT EXISTS short_description_override_indo TEXT;

-- ---------------------------------------------------------------------------
-- Indonesian translations for the bounded label sets (tags + rating ranges).
-- Per-cafe free text (location.description, cafe_review.content, etc.) is
-- authored separately; until then the English baseline shows via fallback.
-- ---------------------------------------------------------------------------

UPDATE tag SET name_indo = v.name_indo, description_indo = v.description_indo
FROM (VALUES
  ('wfc-friendly', 'Cocok buat Kerja dari Kafe (WFC)', 'Kafe yang cocok untuk Work From Cafe (WFC) adalah kedai kopi yang menyediakan lingkungan kondusif untuk bekerja jarak jauh. Kafe seperti ini biasanya menawarkan tempat duduk yang nyaman, banyak colokan listrik, wifi yang andal, dan suasana tenang yang memungkinkan pelanggan fokus mengerjakan tugasnya sambil menikmati kopi.'),
  ('reading', 'Cocok buat membaca', 'Kafe dengan lingkungan ramah membaca dirancang untuk pelanggan yang gemar membaca sambil menyeruput kopi. Kafe seperti ini sering menyediakan tempat duduk yang nyaman, pencahayaan yang baik, dan suasana tenang sehingga pelanggan dapat tenggelam dalam bacaannya.'),
  ('city-view', 'Pemandangan Kota', 'Kafe dengan pemandangan kota menawarkan kesempatan menikmati kopi sambil menyaksikan indahnya lanskap Kota Bandung. Kafe seperti ini biasanya berada di perbukitan atau gedung bertingkat.'),
  ('open-24-hours', 'Buka 24 Jam', 'Kafe yang buka 24 jam memberi kenyamanan untuk menikmati kopi kapan saja, siang maupun malam. Kafe ini cocok untuk berbagai pelanggan, termasuk pecinta malam, mahasiswa yang begadang mengerjakan tugas, dan siapa pun yang butuh asupan kafein larut malam.'),
  ('pet-friendly', 'Ramah Hewan Peliharaan', 'Kafe ramah hewan peliharaan menyambut pelanggan yang ingin membawa hewan kesayangannya sambil menikmati kopi. Kafe ini sering menyediakan fasilitas seperti mangkuk air, camilan hewan, dan area duduk khusus bagi pelanggan yang membawa peliharaan.'),
  ('comfortable-prayer-room', 'Mushola Nyaman', 'Kafe dengan mushola yang nyaman menyediakan ruang khusus bagi pelanggan untuk beribadah di lingkungan yang bersih, tenang, dan nyaman. Kafe ini cocok bagi pelanggan yang ingin tetap menjalankan ibadah sambil menikmati kopi.'),
  ('live-music', 'Live Musik', 'Kafe dengan live musik menawarkan kesempatan menikmati kopi sambil mendengarkan penampilan musisi atau band lokal. Kafe ini biasanya memiliki panggung atau area khusus pertunjukan, menciptakan suasana yang hidup dan menghibur.'),
  ('air-conditioned-seating', 'Ruangan Ber-AC', 'Kafe dengan ruangan ber-AC memberikan lingkungan yang nyaman untuk menikmati kopi, terutama saat cuaca panas atau lembap. Kafe ini biasanya memiliki pendingin udara yang menjaga suhu tetap sejuk sehingga pelanggan dapat bersantai dengan nyaman.'),
  ('indoor-smoking', 'Bisa Merokok di Dalam', 'Kafe yang memperbolehkan merokok/vape di dalam menyediakan area khusus bagi pelanggan untuk merokok/vape sambil menikmati kopi. Kafe ini biasanya memiliki ventilasi yang baik agar asapnya tidak mengganggu pelanggan yang tidak merokok.'),
  ('kalcer', 'Kalcer', 'Kafe yang dipenuhi anak muda Bandung dengan gaya ''skena''. Pakai outfit ''kalcer'' terbaikmu agar bisa membaur, siapa tahu kamu bisa mengobrol dengan ''teteh-teteh Bandung'' yang sudah lama kamu incar.'),
  ('aesthetic', 'Estetik', 'Kafe dengan suasana estetik dirancang untuk memberikan atmosfer yang indah dan Instagramable. Kafe ini sering menampilkan desain interior yang unik dan bergaya, dekorasi kreatif, serta perhatian pada detail yang menciptakan pengalaman berkesan.'),
  ('grab-n-go', 'Grab N Go', 'Kafe yang menawarkan opsi cepat dan praktis untuk pelanggan yang sedang dalam perjalanan atau terburu-buru. Kafe seperti ini biasanya memiliki menu yang disederhanakan dengan item siap saji, layanan yang efisien, dan fokus pada kecepatan tanpa mengorbankan kualitas, menjadikannya ideal untuk pelanggan yang membutuhkan kopi cepat atau makanan ringan saat dalam perjalanan ke tempat kerja atau aktivitas lainnya.'),
  ('perfect-for-date', 'Cocok buat ngedate', 'Kafe yang sempurna untuk kencan menyediakan suasana romantis dan intim bagi pasangan untuk menikmati kopi bersama. Kafe seperti ini sering memiliki pengaturan tempat duduk yang nyaman, pencahayaan redup, dan suasana yang menawan yang menciptakan pengalaman yang berkesan dan menyenangkan bagi pasangan pada kencan mereka.')
) AS v(slug, name_indo, description_indo)
WHERE tag.slug = v.slug;

UPDATE rating_category SET name_indo = v.name_indo, short_description_indo = v.short_description_indo
FROM (VALUES
  ('price-rank', 0::numeric,     'Bandung',            'Murah, seperti seharusnya harga kopi di Bandung'),
  ('price-rank', 25001::numeric, 'Riau',               'Menengah, tapi wajar karena lokasinya di pusat kota'),
  ('price-rank', 45001::numeric, 'Jakarta',            'Mahal, ini di Bandung tapi harganya seperti kafe di Jakarta'),
  ('vibe',       0::numeric,     'Nongkrong',          'Lebih ramai dan cocok untuk nongkrong bareng teman'),
  ('vibe',       1.68::numeric,  'Serbaguna',          'Cocok untuk segala suasana, baik bekerja, nongkrong, atau keduanya sekaligus'),
  ('vibe',       3.34::numeric,  'Nyaman',             'Lingkungan nyaman dan cozy, tempat ideal untuk bersantai menikmati kopi'),
  ('noise',      0::numeric,     'Tenang',             'Cocok untuk kerja fokus atau sesi membaca'),
  ('noise',      1.68::numeric,  'Sedang',             'Seimbang antara suasana ramai dan tenang, cocok untuk berbagai keperluan'),
  ('noise',      3.34::numeric,  'Ramai',              'Lebih cocok untuk ngobrol atau ngeghibah bareng teman dan tertawa lepas'),
  ('wifi',       0::numeric,     'Sangat lambat',      'Kurang cocok untuk bekerja, mending ngobrol sama temen'),
  ('wifi',       1.68::numeric,  'Standar',            'Cukup untuk menyelesaikan pekerjaanmu'),
  ('wifi',       3.34::numeric,  'Cepat',              'Cocok untuk video call, mengunduh file besar, dan semua kebutuhan kerja'),
  ('meals',      0::numeric,     'Sedikit',            'Camilan ringan untuk menemani kopimu di sore hari'),
  ('meals',      1.68::numeric,  'Standar',            'Seperti kafe pada umumnya, pas jika kamu bukan pemakan besar'),
  ('meals',      3.34::numeric,  'Mengenyangkan',      'Siap-siap kenyang dengan porsi makanannya, lebih dari sekadar camilan'),
  ('atmosphere', 0::numeric,     'Tenang & Alami',     'Banyak tanaman hijau, udara segar, dan suasana tenang. Cocok untuk bersantai'),
  ('atmosphere', 1.68::numeric,  'Seimbang',           'Perpaduan kenyamanan dan energi. Cocok untuk berbagai suasana hati'),
  ('atmosphere', 3.34::numeric,  'Urban & Energetik',  'Ramai, bergaya, dan sosial. Tempat untuk melihat dan dilihat'),
  ('parking',    0::numeric,     'Terbatas',           'Hanya tersedia sedikit tempat parkir'),
  ('parking',    1.68::numeric,  'Normal',             'Cukup ruang untuk setiap pelanggan'),
  ('parking',    3.34::numeric,  'Mudah',              'Banyak tempat parkir, tidak perlu khawatir mencari tempat')
) AS v(type, lower_bound, name_indo, short_description_indo)
WHERE rating_category.type = v.type::rating_category_type_enum
  AND rating_category.lower_bound = v.lower_bound;
