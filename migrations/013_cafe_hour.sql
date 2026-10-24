-- Per-day opening hours (BCF-19). Additive: replaces cafe.open_hour / close_hour,
-- which stay in place until 014_drop_cafe_open_hour.sql.
--
--   psql -U postgres -d bandung_coffeeshop -f migrations/013_cafe_hour.sql
--
-- Which rows apply to a date, and how they resolve, is defined once in
-- CLAUDE.md "Opening hours" (shared by be, admin and scraper).
\set ON_ERROR_STOP on

begin;

create type day_enum as enum ('mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun');

alter type day_enum owner to bdgcafe;

-- day null = every day. valid_from / valid_until null on both = regular hours;
-- otherwise a dated row (holiday / special event), valid_until inclusive.
-- close = true marks the day closed and its hours are ignored.
-- close_hour < open_hour = overnight; 24 hours = 00:00–23:59.
create table cafe_hour
(
    id          serial
        primary key,
    cafe_id     integer                 not null
        references cafe
            on delete cascade,
    day         day_enum,
    open_hour   time,
    close_hour  time,
    valid_from  date,
    valid_until date,
    close       boolean   default false not null,
    created_at  timestamp default now() not null,
    updated_at  timestamp default now() not null
);

alter table cafe_hour
    owner to bdgcafe;

create index idx_cafe_hour_cafe
    on cafe_hour (cafe_id);

-- Backfill: the single pair becomes one regular, every-day row. Cafes with
-- only one side set are skipped (their hours become unknown).
insert into cafe_hour (cafe_id, open_hour, close_hour)
select id, open_hour, close_hour
from cafe
where open_hour is not null
  and close_hour is not null;

commit;
