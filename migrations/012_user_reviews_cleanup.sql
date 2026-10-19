-- User reviews, part 2 of 2 (BCF-29). Run right AFTER the user-reviews BE and
-- admin are deployed (011 must already have run). Until this file runs,
-- cafe_rating keeps UNIQUE (cafe_id, category_type), so a second reviewer's
-- ratings on an already-rated cafe fail to insert — run it promptly.
--
--   psql -U postgres -d bandung_coffeeshop -f migrations/012_user_reviews_cleanup.sql
\set ON_ERROR_STOP on

begin;

-- 011 copied each cafe's latest review text into cafe.description; drop the
-- duplicate from the review. Rule: clear content only where it is still an
-- exact copy of the cafe's about text, so nothing written since 011 (by users,
-- or an edited about) is lost. Superseded legacy reviews (status 'deleted' by
-- 011) keep their text for history; they are never served.
update cafe_review rv
set content = null
from cafe c
where c.id = rv.cafe_id
  and rv.content = c.description;

alter table cafe_review
    drop column is_subjective,
    drop column personal_score,
    drop column content_indo;

-- Ratings written between 011 and this file by a client that did not set
-- review_id are attached to the cafe's top visible review (pinned first). Any
-- row that still has no review is a data error: abort instead of deleting it.
update cafe_rating r
set review_id = top.id
from (select distinct on (cafe_id) cafe_id, id
      from cafe_review
      where status = 'approved'
      order by cafe_id, is_pinned desc, updated_at desc, id desc) top
where r.review_id is null
  and top.cafe_id = r.cafe_id;

do
$$
    begin
        if exists (select 1 from cafe_rating where review_id is null) then
            raise exception 'cafe_rating rows without a review remain; attach or delete them, then re-run';
        end if;
    end
$$;

-- Ratings are per review now: one score per category per review.
alter table cafe_rating
    alter column review_id set not null,
    drop constraint cafe_rating_cafe_id_category_type_key,
    add constraint cafe_rating_review_id_category_type_key
        unique (review_id, category_type);

commit;
