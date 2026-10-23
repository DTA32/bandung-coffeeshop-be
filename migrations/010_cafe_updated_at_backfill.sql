-- "Last updated" on the cafe detail page reads cafe.updated_at (BCF-35).
-- Until now the page showed the latest review's updated_at, so align
-- cafe.updated_at with it once: every cafe that has a review takes the
-- updated_at of its latest review, picked with the same ordering the
-- /v1/cafe/:id/review API used (COALESCE(visited_at, created_at::date) DESC,
-- id DESC). Cafes without a review keep their current value.
-- Run before deploying the BE that exposes updated_at on /v1/cafe/:id.
begin;

update cafe c
set updated_at = lr.updated_at
from (select distinct on (cafe_id) cafe_id, updated_at
      from cafe_review
      order by cafe_id, coalesce(visited_at, created_at::date) desc, id desc) lr
where lr.cafe_id = c.id;

commit;
