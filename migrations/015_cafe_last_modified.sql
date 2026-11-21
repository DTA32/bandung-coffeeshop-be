-- Derived cafe last-modified time (BCF-44). Additive: run before deploying the
-- BE that reads it; the current BE and admin keep working without it.
--
--   psql -U postgres -d bandung_coffeeshop -f migrations/015_cafe_last_modified.sql
--
-- cafe_last_modified is the only definition of a cafe's "last updated" time
-- (cafe detail updated_at, the "Recently updated" sort, sitemap lastmod).
-- cafe.updated_at now means only "the cafe row changed". GREATEST skips NULLs,
-- so a cafe with no child rows falls back to its cafe / location timestamps.
-- Deleted child rows don't count, so the value can move backwards. See
-- CLAUDE.md "Last modified".
\set ON_ERROR_STOP on

begin;

create view cafe_last_modified as
select c.id as cafe_id,
       greatest(c.updated_at, l.updated_at, p.updated_at,
                h.at, f.at, rv.at, t.at, i.at) as last_modified
from cafe c
join location l on l.id = c.location_id
left join cafe_price p on p.cafe_id = c.id
left join lateral (select max(updated_at) at from cafe_hour where cafe_id = c.id) h on true
left join lateral (select max(published_at) at from cafe_feed where cafe_id = c.id and visible) f on true
left join lateral (select max(updated_at) at from cafe_review where cafe_id = c.id and status = 'approved') rv on true
left join lateral (select max(created_at) at from cafe_tag where cafe_id = c.id) t on true
left join lateral (select max(updated_at) at from location_image where location_id = c.location_id) i on true;

alter view cafe_last_modified
    owner to bdgcafe;

-- One per lateral above, so each max() is an index lookup.
create index idx_cafe_hour_cafe_updated
    on cafe_hour (cafe_id asc, updated_at desc);

create index idx_cafe_feed_cafe_updated
    on cafe_feed (cafe_id asc, published_at desc)
    where visible;

create index idx_cafe_review_cafe_updated
    on cafe_review (cafe_id asc, updated_at desc)
    where status = 'approved';

create index idx_cafe_tag_cafe_created
    on cafe_tag (cafe_id asc, created_at desc);

create index idx_location_image_location_updated
    on location_image (location_id asc, updated_at desc);

-- Covered by idx_cafe_hour_cafe_updated.
drop index idx_cafe_hour_cafe;

-- The sort is on the derived value now (sorted in memory; a plain view can't
-- be indexed).
drop index idx_cafe_featured_updated;

commit;
