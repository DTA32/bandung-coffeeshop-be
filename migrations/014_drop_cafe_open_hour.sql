-- Per-day opening hours, cleanup (BCF-19). Apply ONLY after be, admin and
-- scraper are all deployed on cafe_hour (013 must already have run): until
-- then the old builds still read / write these columns.
--
--   psql -U postgres -d bandung_coffeeshop -f migrations/014_drop_cafe_open_hour.sql
\set ON_ERROR_STOP on

begin;

alter table cafe
    drop column open_hour,
    drop column close_hour;

commit;
