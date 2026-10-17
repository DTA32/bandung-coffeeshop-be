-- Contact fields kept fresh by bandung-coffeeshop-scraper (BCF-11 / BCF-12).
-- phone: normalised to +62… by the scraper; menu: URL of the cafe's menu
-- (Drive / Canva / PDF / Linktree link). The address stays in location.description.
alter table cafe
    add column phone text,
    add column menu  text;
