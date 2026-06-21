-- Consolidated SQL data seeder.
-- Merges the former 004 (tags + rating categories + sample cafe data),
-- 006 (Indonesian i18n translations), 007 (rating long descriptions), and
-- 008's data (rating_type_label rows). All DDL those files used to carry now
-- lives in 001_init.sql; this file is data only.
--
-- Run order: 001_init.sql -> 002_cafe_seeder.py -> 003_data_seeder.sql.
-- Depends on 002 having seeded the `cafe`/`location` rows (the sample
-- cafe_rating/cafe_tag/cafe_price below reference cafe_id 85). Sections are
-- ordered so foreign keys resolve: rating_type_label first (FK target for
-- rating_category.type and cafe_rating.category_type), then the base label
-- sets, then the per-cafe sample rows, then the localized UPDATEs.

-- ---------------------------------------------------------------------------
-- 1. rating_type_label: canonical rating-type registry + localized labels.
--    price-rank is included because cafe_rating / rating_category carry
--    price-rank rows (the cafe-detail ratings card renders it) even though
--    /v1/filters surfaces it as price tiers, not a rating group.
-- ---------------------------------------------------------------------------
INSERT INTO rating_type_label (type, label, label_indo) VALUES
    ('price-rank', 'Price Rank',       'Peringkat Harga'),
    ('vibe',       'Vibe',             'Suasana'),
    ('noise',      'Noise Level',      'Tingkat Kebisingan'),
    ('wifi',       'Wifi Speed',       'Kecepatan Wifi'),
    ('meals',      'Meals Generosity', 'Porsi Makanan'),
    ('atmosphere', 'Atmosphere',       'Atmosfer'),
    ('parking',    'Parking',          'Parkir')
ON CONFLICT (type) DO NOTHING;

-- ---------------------------------------------------------------------------
-- 2. tags
-- ---------------------------------------------------------------------------
INSERT INTO tag (name, description, slug) VALUES
('Work From Cafe (WFC) Friendly', 'A Work From Cafe (WFC) friendly cafe is a coffee shop that provides an environment conducive to remote work. These cafes typically offer amenities such as comfortable seating, ample power outlets, reliable wifi, and a quiet atmosphere that allows customers to focus on their tasks while enjoying their coffee.', 'wfc-friendly'),
('Reading', 'A cafe with a reading-friendly environment is designed to cater to customers who enjoy reading while sipping their coffee. These cafes often provide cozy seating arrangements, good lighting, and a quiet ambiance that allows customers to immerse themselves in their books or other reading materials.', 'reading'),
('City View', 'A cafe with a city view offers customers the opportunity to enjoy their coffee while taking in the sights of beautiful Bandung cityscape. These cafes are often located on hills or high-rise buildings', 'city-view'),
('Open 24 Hours', 'A cafe that is open 24 hours provides customers with the convenience of enjoying their coffee at any time of the day or night. These cafes cater to a wide range of customers, including night owls, university students pulling all-nighters on their next-day assignments, and those seeking a late-night caffeine fix.', 'open-24-hours'),
('Pet Friendly', 'A pet-friendly cafe welcomes customers who wish to bring their pets along while enjoying their coffee. These cafes often provide amenities such as water bowls, pet treats, and designated seating areas for customers with pets.', 'pet-friendly'),
('Comfortable Prayer Room', 'A cafe with a comfortable prayer room provides a dedicated space for customers to perform their prayers in a clean, quiet, and comfortable environment. These cafes often cater to customers who observe religious practices and want to ensure they can fulfill their spiritual needs while enjoying their coffee.', 'comfortable-prayer-room'),
('Live Music', 'A cafe that features live music offers customers the opportunity to enjoy their coffee while listening to performances by local musicians or bands. These cafes often have a stage or designated area for live performances, creating a vibrant and entertaining atmosphere for customers.', 'live-music'),
('Air-conditioned Seating', 'A cafe with air-conditioned seating provides customers with a comfortable environment to enjoy their coffee, especially in hot or humid climates. These cafes typically have air conditioning units that help regulate the temperature, ensuring that customers can relax and enjoy their beverages without discomfort.', 'air-conditioned-seating'),
('Indoor Smoking', 'A cafe that allows indoor smoking provides a designated area where customers can smoke while enjoying their coffee. These cafes often have proper ventilation systems to ensure that the smoke does not affect non-smoking customers, creating a space for smokers to enjoy their coffee without restrictions.', 'indoor-smoking'),
('Kalcer', 'A cafe that is filled with Bandung youths with ''skena'' outfit and style, use your best ''kalcer'' outfit to blend in and you might be able to chat with that ''teteh-teteh Bandung'' that you''ve been eyeing for a while', 'kalcer'),
('Aesthetic', 'A cafe with an aesthetic environment is designed to provide customers with a visually pleasing and Instagram-worthy atmosphere. These cafes often feature unique and stylish interior designs, creative decor, and attention to detail that creates a memorable experience for customers.', 'aesthetic'),
('Unique Concept', 'A cafe with a unique concept stands out from the typical coffee shop by offering a distinctive theme or experience. These cafes often have creative interior designs, innovative menu offerings, or interactive elements that set them apart and provide customers with a memorable and one-of-a-kind coffee experience.', 'unique-concept'),
('Hidden Gem', 'A hidden gem cafe is a lesser-known coffee shop that offers exceptional quality and a unique atmosphere. These cafes may be tucked away in less frequented areas or have a low profile, but they provide customers with a delightful and often surprising coffee experience that is worth seeking out.', 'hidden-gem'),
('Rooftop', 'A cafe with a rooftop offers customers the opportunity to enjoy their coffee while taking in panoramic views of the surrounding area. These cafes often have outdoor seating on the rooftop, creating a unique and enjoyable atmosphere for customers to relax and savor their beverages.', 'rooftop'),
('Vegan/Vegetarian Options', 'A cafe that offers vegan or vegetarian options provides customers with plant-based food and beverage choices. These cafes cater to customers who follow a vegan or vegetarian lifestyle, ensuring that they can enjoy delicious meals and drinks that align with their dietary preferences while enjoying their coffee.', 'vegan-vegetarian-options'),
('Specialty Coffee', 'A cafe that serves specialty coffee offers high-quality, expertly crafted coffee beverages. These cafes often source their coffee beans from specific regions, use precise brewing methods, and have skilled baristas who create unique and flavorful coffee experiences for customers.', 'specialty-coffee'),
('Family-friendly', 'A family-friendly cafe caters to customers with children by providing amenities such as high chairs, play areas, and kid-friendly menu options. These cafes create a welcoming environment for families to enjoy their coffee together while ensuring that the needs of both parents and children are met.', 'family-friendly'),
('Artistic Vibe', 'A cafe with an artistic vibe is designed to inspire creativity and appreciation for the arts. These cafes often feature local artwork, host art events or workshops, and create an atmosphere that encourages customers to express themselves and connect with the artistic community while enjoying their coffee.', 'artistic-vibe');

-- ---------------------------------------------------------------------------
-- 3. rating categories (buckets). type references rating_type_label(type).
-- ---------------------------------------------------------------------------
INSERT INTO rating_category (name, short_description, long_description, slug, type, lower_bound, upper_bound) VALUES
('Bandung', 'Cheap, like what bandung coffee shop should be', '', 'bandung', 'price-rank', 0, 25000),
('Riau', 'Mid-range, but it''s understandable since it''s located in the city center', '', NULL, 'price-rank', 25001, 45000),
('Jakarta', 'Expensive, we''re in Bandung but the price is like Jakarta coffee shop', '', NULL,'price-rank', 45001, 999999),
('Hangout', 'More lively and suitable for hanging out with friends', '', 'hangout', 'vibe', 0, 1.67),
('All-rounder', 'Good for all occasions, whether it''s for working, hanging out, or just enjoying a cup of coffee', '', 'all-rounder', 'vibe', 1.68, 3.33),
('Comfy', 'Comfortable and cozy environment, making it an ideal place to relax and enjoy your coffee', '', 'comfortable', 'vibe', 3.34, 5),
('Quiet', 'Perfect for deep-work or reading session', '', 'quiet', 'noise', 0, 1.67),
('Moderate', 'Balance between a lively atmosphere and a quiet environment, making it suitable for various occasions', '', NULL, 'noise', 1.68, 3.33),
('Loud', 'More suitable for hanging out with your friends and let all the laughter out', '', NULL, 'noise', 3.34, 5),
('Very slow', 'Not really good for working, i''d suggest you to just chat with your friends', '', NULL, 'wifi', 0, 1.67),
('Average', 'Just enough for you to get your work done', '', NULL, 'wifi', 1.68, 3.33),
('Fast', 'Good for video calls, downloading large files, and all of your work needs', '', 'fast', 'wifi', 3.34, 5),
('Tiny', 'Quick bites for afternoon snack or to accompany your coffee', '', NULL, 'meals', 0, 1.67),
('Average', 'Just like any other cafe, might be perfect if you''re not a big eater', '', NULL, 'meals', 1.68, 3.33),
('Generous', 'Expect to full your stomach with the meals they provide, they''re more than just a snack', '', 'generous', 'meals', 3.34, 5),
('Calm & Natural','Lots of greenery, fresh air, and a quiet atmosphere. Perfect for unwinding', '', 'natural','atmosphere', 0,1.67),
('Balanced','A mix of comfort and energy. Fits most moods and occasions', '', NULL,'atmosphere', 1.68, 3.33),
('Urban & Energetic', 'Lively, stylish, and social. The place to see and be seen','', 'urban', 'atmosphere', 3.34, 5),
('Limited', 'Only few parking spots available', '', NULL, 'parking', 0, 1.67),
('Normal', 'Enough space for every customer', '', NULL, 'parking', 1.68, 3.33),
('Easy', 'Plenty of parking space, no need to worry about finding a spot', '', 'easy', 'parking', 3.34, 5);

-- cafe_rating, cafe_tag, and cafe_price example, using accio-coffee as an example cafe
INSERT INTO cafe_rating (cafe_id, category_type, score) VALUES
(85, 'price-rank', 23000),
(85, 'vibe', 4.5),
(85, 'noise', 1),
(85, 'wifi', 3),
(85, 'meals', 1),
(85, 'atmosphere', 1),
(85, 'parking', 5);

INSERT INTO cafe_tag (cafe_id, tag_id) VALUES
(85, 1),
(85, 2);

INSERT INTO cafe_price (cafe_id, price_range_min, price_range_max, coffee_price_min, coffee_price_max, snack_price_min, snack_price_max, food_price_min, food_price_max) VALUES
(85, 18000, 28000, 20000, 28000, 15000, 17000, 25000, 30000);

-- ---------------------------------------------------------------------------
-- 4. Indonesian translations for the bounded label sets (tags + rating ranges).
--    The English columns seeded above are the baseline/fallback; queries prefer
--    the *_indo column when lang='id' and fall back to English when it is empty.
--    Per-cafe free text is authored separately. (was 006_i18n.sql)
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
WHERE rating_category.type = v.type
  AND rating_category.lower_bound = v.lower_bound;

-- ---------------------------------------------------------------------------
-- 5. rating_category long descriptions (EN + ID). These power the explore SRP
--    blurb (ExploreBlurb) for rating-bucket pages. Matched by (type,
--    lower_bound) so it's environment-agnostic. (was 007_rating_long_description.sql)
-- ---------------------------------------------------------------------------
UPDATE rating_category AS rc
SET long_description = v.long_description,
    long_description_indo = v.long_description_indo,
    updated_at = NOW()
FROM (VALUES
  -- price-rank
  ('price-rank', 0,
   'Bandung-priced cafes keep things wonderfully affordable, with most drinks and snacks landing under 25k — exactly how a Bandung coffee shop should be. Great for students, daily caffeine runs, or long work-from-cafe sessions without emptying your wallet.',
   'Kafe dengan harga khas Bandung tetap ramah di kantong, kebanyakan minuman dan camilan di bawah 25 ribu — persis seperti seharusnya harga kopi di Bandung. Cocok untuk mahasiswa, ngopi harian, atau sesi kerja dari kafe yang panjang tanpa mengeringkan dompet.'),
  ('price-rank', 25001,
   'Mid-range cafes around the 25k–45k mark sit comfortably in the middle of price ranges, a fair trade-off for a more central location or a more polished space. A solid pick when you want quality coffee and a nicer setting in Bandung without going all-out.',
   'Kafe dengan harga kisaran 25–45 ribu berada di tengah-tengah rentang harga, wajar untuk lokasi yang lebih sentral atau suasana yang lebih rapi. Pilihan pas saat kamu mau kopi berkualitas dan tempat yang lebih nyaman di Bandung tanpa harus mahal-mahal.'),
  ('price-rank', 45001,
   'These are the priciest coffee shops in Bandung, with bills that feel more like Jakarta than Bandung. Expect premium beans, standout interiors, or a prime location — best saved for a treat or a special hangout.',
   'Ini kafe-kafe yang cukup mahal di Bandung, dengan harga yang terasa lebih seperti kafe Jakarta ketimbang Bandung. Siap-siap dengan biji kopi premium, interior yang keren, atau lokasi strategis — paling pas untuk sesekali memanjakan diri atau nongkrong spesial.'),

  -- vibe
  ('vibe', 0,
   'Hangout cafes in Bandung lean lively and social, made for catching up with friends over coffee. Expect an energetic buzz, plenty of group seating, and an easygoing mood that''s better for conversation than concentration.',
   'Kafe dengan vibe nongkrong di Bandung cenderung ramai dan sosial, pas untuk ngobrol bareng teman sambil ngopi. Suasananya energik, banyak tempat duduk untuk rame-rame, dan moodnya santai — lebih cocok untuk ngobrol daripada fokus kerja.'),
  ('vibe', 1.68,
   'All-rounder cafes strike a balanced vibe that works for almost anything — getting work done, hanging out with friends, or simply enjoying a cup of coffee on your own. A safe, versatile choice in Bandung when you''re not sure what the day calls for.',
   'Kafe serbaguna punya vibe seimbang yang cocok untuk hampir segalanya — bekerja, nongkrong bareng teman, atau sekadar menikmati kopi sendirian. Pilihan aman dan fleksibel di Bandung saat kamu belum yakin mau suasana seperti apa.'),
  ('vibe', 3.34,
   'Comfy cafes wrap you in a cozy, relaxed environment that invites you to slow down and stay a while. With soft seating and a calming mood, they''re an ideal spot in Bandung to unwind, read, or savor your coffee without rushing.',
   'Kafe yang nyaman menghadirkan lingkungan cozy dan santai yang bikin betah berlama-lama. Dengan tempat duduk empuk dan suasana menenangkan, ini tempat ideal di Bandung untuk bersantai, membaca, atau menikmati kopi tanpa terburu-buru.'),

  -- noise
  ('noise', 0,
   'Quiet cafes keep noise to a minimum, making them perfect for deep work, studying, or a focused reading session in Bandung. If you need to concentrate over a good cup of coffee with few distractions, this is your spot.',
   'Kafe yang tenang menjaga kebisingan seminimal mungkin, pas untuk kerja fokus, belajar, atau sesi membaca di Bandung. Kalau kamu butuh konsentrasi sambil menikmati kopi tanpa banyak gangguan, ini tempatnya.'),
  ('noise', 1.68,
   'Moderate-noise cafes balance a lively atmosphere with a calm-enough environment, so you can chat with friends or get light work done. A flexible middle ground that suits most occasions in Bandung.',
   'Kafe dengan kebisingan sedang menyeimbangkan suasana ramai dengan lingkungan yang cukup tenang, jadi kamu bisa ngobrol dengan teman atau mengerjakan tugas ringan. Titik tengah yang fleksibel untuk berbagai keperluan di Bandung.'),
  ('noise', 3.34,
   'Loud cafes are buzzing and full of energy — better suited for hanging out, laughing with friends, and letting loose than for quiet work. Come for the lively social atmosphere, not the focus.',
   'Kafe yang ramai nan penuh energi — lebih cocok untuk nongkrong, tertawa bareng teman, dan bersantai daripada kerja yang butuh ketenangan. Datang untuk suasana sosial yang hidup, bukan untuk fokus.'),

  -- wifi
  ('wifi', 0,
   'Wifi here is very slow, so it''s not the best choice for working — you''ll be happier chatting with friends than fighting a weak connection. Come for the coffee and the company, not the work-from-cafe setup.',
   'Wifi di sini sangat lambat, jadi kurang cocok untuk kerja — kamu bakal lebih senang ngobrol dengan teman daripada berjuang dengan koneksi lemah. Datang untuk kopi dan kebersamaan, bukan untuk kerja dari kafe.'),
  ('wifi', 1.68,
   'Average wifi is just enough to get everyday work done — emails, browsing, and light tasks all run fine. A dependable pick for a casual work-from-cafe session in Bandung.',
   'Wifi standar cukup untuk menyelesaikan pekerjaan sehari-hari — email, browsing, dan tugas ringan berjalan lancar. Pilihan andal untuk sesi kerja dari kafe yang santai di Bandung.'),
  ('wifi', 3.34,
   'Fast wifi makes these some of the best work-from-cafe spots in Bandung — smooth video calls, large downloads, and heavy multitasking all handled with ease. Ideal for remote workers and students who need a reliable connection over coffee.',
   'Wifi cepat menjadikan tempat-tempat ini salah satu spot kerja dari kafe terbaik di Bandung — video call lancar, unduh file besar, dan multitasking berat semua beres tanpa hambatan. Ideal untuk pekerja remote dan mahasiswa yang butuh koneksi andal sambil ngopi.'),

  -- meals
  ('meals', 0,
   'Expect tiny portions here — quick bites and light snacks to accompany your coffee rather than a full meal. Perfect for an afternoon nibble, not for showing up hungry.',
   'Porsi di sini kecil — camilan ringan untuk menemani kopimu, bukan makan besar. Pas untuk ngemil sore, bukan untuk datang dalam keadaan lapar.'),
  ('meals', 1.68,
   'Meals here are about average — much like most cafes, enough to take the edge off if you''re not a big eater. A reasonable option when you want a snack alongside your coffee.',
   'Porsi makanannya standar — seperti kebanyakan kafe, cukup mengganjal kalau kamu bukan pemakan besar. Opsi yang masuk akal saat kamu ingin camilan menemani kopi.'),
  ('meals', 3.34,
   'Generous portions mean you''ll leave full — the food here is more than just a snack and can easily stand in for a proper meal. Great for cafes in Bandung where you want good coffee and a satisfying plate in one stop.',
   'Porsi yang mengenyangkan bikin kamu pulang dalam keadaan kenyang — makanannya lebih dari sekadar camilan dan bisa jadi pengganti makan berat. Pas untuk kafe di Bandung saat kamu mau kopi enak sekaligus makanan yang mengenyangkan.'),

  -- atmosphere
  ('atmosphere', 0,
   'Calm and natural cafes are full of greenery, fresh air, and a peaceful mood — a refreshing escape from the city. Perfect for unwinding in Bandung when you want nature, quiet, and a slow cup of coffee.',
   'Kafe yang tenang dan alami penuh tanaman hijau, udara segar, dan suasana damai — pelarian menyegarkan dari hiruk-pikuk kota. Pas untuk bersantai di Bandung saat kamu mencari nuansa alam, ketenangan, dan kopi yang dinikmati pelan-pelan.'),
  ('atmosphere', 1.68,
   'Balanced cafes mix comfort and energy in just the right measure, fitting most moods and occasions. Whether you''re working, meeting up, or relaxing, this versatile atmosphere adapts to the day.',
   'Kafe dengan atmosfer seimbang memadukan kenyamanan dan energi dalam takaran pas, cocok untuk berbagai suasana hati dan keperluan. Mau kerja, ketemuan, atau bersantai, atmosfer serbaguna ini menyesuaikan dengan harimu.'),
  ('atmosphere', 3.34,
   'Urban and energetic cafes are lively, stylish, and social — the kind of place to see and be seen in Bandung. Come for the buzzing crowd, modern design, and a vibrant scene that''s as much about the atmosphere as the coffee.',
   'Kafe yang urban dan energetik itu ramai, bergaya, dan sosial — tempat untuk melihat dan dilihat di Bandung. Datang untuk keramaian, desain modern, dan suasana hidup yang sama menariknya dengan kopinya.'),

  -- parking
  ('parking', 0,
   'Parking is limited here, with only a few spots available — worth planning ahead, or coming by motorbike or ride-hailing. Best for quick visits rather than big group meetups.',
   'Tempat parkir di sini terbatas, hanya tersedia beberapa tempat — sebaiknya rencanakan dulu, atau datang naik motor atau ojek online. Lebih cocok untuk kunjungan singkat daripada kumpul rame-rame.'),
  ('parking', 1.68,
   'Parking here is normal — enough space for the usual flow of customers without much fuss. You should find a spot without too much hassle on most visits.',
   'Parkir di sini normal — cukup ruang untuk pelanggan pada umumnya tanpa banyak masalah. Kamu biasanya bisa dapat tempat tanpa terlalu repot.'),
  ('parking', 3.34,
   'Easy parking means plenty of space and no stress about finding a spot — ideal if you''re driving in Bandung or coming with friends. Pull up, park, and head straight for the coffee.',
   'Parkir mudah berarti banyak tempat dan tanpa pusing cari tempat — ideal kalau kamu bawa mobil di Bandung atau datang bareng teman. Tinggal datang, parkir, dan langsung menikmati kopi.')
) AS v(type, lower_bound, long_description, long_description_indo)
WHERE rc.type = v.type
  AND rc.lower_bound = v.lower_bound;
