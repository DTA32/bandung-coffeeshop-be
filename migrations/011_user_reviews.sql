-- User reviews, part 1 of 2 (BCF-29). Additive only: run BEFORE deploying the
-- user-reviews BE and admin; the currently deployed BE keeps working on it.
--
--   psql -U postgres -d bandung_coffeeshop -v radit_email=<radit's google email> \
--        -f migrations/011_user_reviews.sql
--
-- radit_email must match a users row (Radit has to have signed in once via
-- Google). Every existing review becomes his: approved and pinned. The
-- migration aborts with an exception when no such user exists.
--
-- Caveat while the OLD admin is still deployed: cafe_review.reviewer_name is
-- NOT NULL after this file, so the old admin can no longer insert reviews
-- (edits still work). Deploy the new admin right after.
--
-- 012_user_reviews_cleanup.sql runs after the new BE + admin are deployed.
\set ON_ERROR_STOP on

begin;

select set_config('bdg.radit_email', :'radit_email', false);

create type review_status_enum as enum ('pending', 'approved', 'rejected', 'deleted');

alter type review_status_enum owner to bdgcafe;

-- "About" blurb for the cafe page; it used to be the review text.
alter table cafe
    add column description      text,
    add column description_indo text;

-- user_id null = partnership review (admin-entered, no account).
alter table cafe_review
    add column user_id            integer references users,
    add column reviewer_name      text,
    add column status             review_status_enum,
    add column is_pinned          boolean default false not null,
    add column external_url_title text,
    add column external_url_link  text;

alter table cafe_rating
    add column review_id integer references cafe_review;

do
$$
    declare
        radit_id   integer;
        radit_name text;
    begin
        select id, name
        into radit_id, radit_name
        from users
        where lower(email) = lower(current_setting('bdg.radit_email'))
        order by id
        limit 1;

        if radit_id is null then
            raise exception 'no users row with email % (pass -v radit_email=... and sign in once first)',
                current_setting('bdg.radit_email');
        end if;

        -- 1. cafe.description(_indo) <- the latest review's text, picked with the
        --    ordering the old /v1/cafe/:id/review API used.
        update cafe c
        set description      = lr.content,
            description_indo = lr.content_indo
        from (select distinct on (cafe_id) cafe_id, content, content_indo
              from cafe_review
              order by cafe_id, coalesce(visited_at, created_at::date) desc, id desc) lr
        where lr.cafe_id = c.id;

        -- 2. Cafes that have ratings but no review row get an empty review so
        --    their ratings can hang off it (no data lost). Scores / content stay
        --    null; created_at/updated_at default to now().
        insert into cafe_review (cafe_id)
        select distinct r.cafe_id
        from cafe_rating r
        where not exists (select 1 from cafe_review rv where rv.cafe_id = r.cafe_id);

        -- 3. Every review becomes Radit's, approved and pinned.
        update cafe_review
        set user_id       = radit_id,
            reviewer_name = radit_name,
            status        = 'approved',
            is_pinned     = true;

        -- 4. Only the latest review per cafe was ever shown. Older ones would
        --    collide on the one-live-review-per-user index below, so they are
        --    soft-deleted (kept for history, never served).
        update cafe_review rv
        set status    = 'deleted',
            is_pinned = false
        where rv.id not in (select distinct on (cafe_id) id
                            from cafe_review
                            order by cafe_id, coalesce(visited_at, created_at::date) desc, id desc);

        -- 5. Ratings belong to the cafe's latest review (every cafe with ratings
        --    has one after step 2, so no review_id is left null).
        update cafe_rating r
        set review_id = lr.id
        from (select distinct on (cafe_id) cafe_id, id
              from cafe_review
              where status <> 'deleted'
              order by cafe_id, coalesce(visited_at, created_at::date) desc, id desc) lr
        where lr.cafe_id = r.cafe_id;
    end
$$;

alter table cafe_review
    alter column reviewer_name set not null,
    alter column status set not null,
    alter column status set default 'pending';

-- One live (non-deleted) review per user per cafe. Partnership reviews
-- (user_id null) are not constrained.
create unique index idx_cafe_review_cafe_user
    on cafe_review (cafe_id, user_id)
    where status <> 'deleted';

-- Serves the paginated review list (keyset on is_pinned, updated_at, id).
create index idx_cafe_review_cafe_visible
    on cafe_review (cafe_id, is_pinned desc, updated_at desc, id desc)
    where status = 'approved';

create index idx_cafe_rating_cafe_type
    on cafe_rating (cafe_id, category_type);

create index idx_cafe_rating_review
    on cafe_rating (review_id);

drop index idx_cafe_review_cafe_latest;

commit;
