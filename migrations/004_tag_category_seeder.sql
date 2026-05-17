INSERT INTO tag (name, description, slug) VALUES
('Work From Cafe (WFC) Friendly', 'A Work From Cafe (WFC) friendly cafe is a coffee shop that provides an environment conducive to remote work. These cafes typically offer amenities such as comfortable seating, ample power outlets, reliable wifi, and a quiet atmosphere that allows customers to focus on their tasks while enjoying their coffee.', 'wfc-friendly'),
('Reading', 'A cafe with a reading-friendly environment is designed to cater to customers who enjoy reading while sipping their coffee. These cafes often provide cozy seating arrangements, good lighting, and a quiet ambiance that allows customers to immerse themselves in their books or other reading materials.', 'reading'),
('City View', 'A cafe with a city view offers customers the opportunity to enjoy their coffee while taking in the sights of beautiful Bandung cityscape. These cafes are often located on hills or high-rise buildings', 'city-view'),
('Open 24 Hours', 'A cafe that is open 24 hours provides customers with the convenience of enjoying their coffee at any time of the day or night. These cafes cater to a wide range of customers, including night owls, university students pulling all-nighters on their next-day assignments, and those seeking a late-night caffeine fix.', 'open-24-hours'),
('Pet Friendly', 'A pet-friendly cafe welcomes customers who wish to bring their pets along while enjoying their coffee. These cafes often provide amenities such as water bowls, pet treats, and designated seating areas for customers with pets.', 'pet-friendly'),
('Comfortable Prayer Room', 'A cafe with a comfortable prayer room provides a dedicated space for customers to perform their prayers in a clean, quiet, and comfortable environment. These cafes often cater to customers who observe religious practices and want to ensure they can fulfill their spiritual needs while enjoying their coffee.', 'prayer-room'),
('Live Music', 'A cafe that features live music offers customers the opportunity to enjoy their coffee while listening to performances by local musicians or bands. These cafes often have a stage or designated area for live performances, creating a vibrant and entertaining atmosphere for customers.', 'live-music'),
('Air-conditioned Seating', 'A cafe with air-conditioned seating provides customers with a comfortable environment to enjoy their coffee, especially in hot or humid climates. These cafes typically have air conditioning units that help regulate the temperature, ensuring that customers can relax and enjoy their beverages without discomfort.', 'air-conditioned'),
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
