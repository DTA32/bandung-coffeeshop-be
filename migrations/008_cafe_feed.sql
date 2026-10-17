-- Per-cafe "Latest updates" entries, created in mission control (usually from an
-- Instagram post) (BCF-11). Single-language content. images holds full
-- image.bdgcafe.com URLs; link is the source (e.g. the Instagram permalink).
-- An entry is shown while visible and (expires_at is null or expires_at > now()).
create table cafe_feed
(
    id           serial
        primary key,
    cafe_id      integer                       not null
        references cafe,
    content      text                          not null,
    images       text[]    default '{}'::text[] not null,
    author       text      default 'Admin'::text not null,
    link         text,
    published_at timestamp default now()       not null,
    expires_at   timestamp,
    visible      boolean   default true        not null,
    created_at   timestamp default now()       not null,
    updated_at   timestamp default now()       not null
);

alter table cafe_feed
    owner to bdgcafe;

create index idx_cafe_feed_cafe
    on cafe_feed (cafe_id asc, published_at desc)
    where visible;
