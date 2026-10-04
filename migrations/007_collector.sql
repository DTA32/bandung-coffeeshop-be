-- Collector tables shared by bandung-coffeeshop-scraper (writes) and
-- bandung-coffeeshop-admin / mission control (reviews and applies) (BCF-11 / BCF-12).

create type collector_source as enum ('gmaps', 'instagram', 'linktree');

alter type collector_source owner to bdgcafe;

-- What to scrape per cafe per source, plus its health. Rows are auto-created by
-- the scraper and controllable from mission control.
--   gmaps:     external_id caches the resolved feature id "0x…:0x…" (from location.gmaps_id cid)
--   instagram: handle is read from cafe.instagram; external_id caches the IG user pk
--   linktree:  external_id is the page URL, auto-discovered from IG bio links when null
create table source_target
(
    cafe_id              integer                 not null
        references cafe,
    source               collector_source        not null,
    external_id          text,
    enabled              boolean   default true  not null,
    last_success_at      timestamp,
    last_error           text,
    consecutive_failures integer   default 0     not null,
    created_at           timestamp default now() not null,
    updated_at           timestamp default now() not null,
    primary key (cafe_id, source)
);

alter table source_target
    owner to bdgcafe;

-- One row per fetch; pruned after 30 days by the scraper.
create table source_snapshot
(
    id         bigserial
        primary key,
    cafe_id    integer                 not null
        references cafe,
    source     collector_source        not null,
    fetched_at timestamp default now() not null,
    raw        jsonb,
    parsed     jsonb,
    hash       text
);

alter table source_snapshot
    owner to bdgcafe;

create index idx_source_snapshot_latest
    on source_snapshot (cafe_id, source, fetched_at desc);

create type proposal_status as enum ('pending', 'approved', 'rejected', 'superseded');

alter type proposal_status owner to bdgcafe;

-- A detected change (field) or a new Instagram post (field = 'ig_post').
-- field: hours, phone, address, status, temp_closed, menu, menu_content, ig_bio, ig_post
create table change_proposal
(
    id             bigserial
        primary key,
    cafe_id        integer                   not null
        references cafe,
    source         collector_source          not null,
    field          text                      not null,
    item_key       text      default ''::text not null, -- '' for fields, shortcode for ig_post
    old_value      jsonb,
    new_value      jsonb,
    evidence_url   text,
    snapshot_id    bigint
        references source_snapshot on delete set null,
    status         proposal_status default 'pending'::proposal_status not null,
    resolved_value jsonb,                               -- value applied / {"cafe_feed_id": n} for ig_post
    created_at     timestamp default now()   not null,
    resolved_at    timestamp
);

alter table change_proposal
    owner to bdgcafe;

-- One pending proposal per field (fields) / per post (ig_post).
create unique index uq_change_proposal_pending
    on change_proposal (cafe_id, field, item_key)
    where (status = 'pending'::proposal_status);

-- A post is proposed at most once, ever (ignored posts never come back).
create unique index uq_change_proposal_post
    on change_proposal (cafe_id, item_key)
    where (field = 'ig_post'::text);

create index idx_change_proposal_queue
    on change_proposal (status, field, created_at);

create table scrape_run
(
    id          bigserial
        primary key,
    source      collector_source        not null,
    started_at  timestamp default now() not null,
    finished_at timestamp,
    ok          integer   default 0     not null,
    failed      integer   default 0     not null,
    proposals   integer   default 0     not null,
    error       text
);

alter table scrape_run
    owner to bdgcafe;

create index idx_scrape_run_source_latest
    on scrape_run (source, started_at desc);
