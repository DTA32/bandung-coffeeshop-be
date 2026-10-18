-- User accounts (BCF-25). Sessions live in Redis.
-- One row per (provider, provider_sub); profile fields are refreshed from the
-- provider on every login. Soft-delete via status = 'deleted' blocks login.
create type user_status_enum as enum ('active', 'deleted');

alter type user_status_enum owner to bdgcafe;

create table users
(
    id           serial
        primary key,
    provider     text                                                not null,
    provider_sub text                                                not null,
    email        text                                                not null,
    name         text                                                not null,
    avatar_url   text,
    status       user_status_enum default 'active'::user_status_enum not null,
    created_at   timestamp        default now()                      not null,
    updated_at   timestamp        default now()                      not null,
    constraint users_provider_sub_key
        unique (provider, provider_sub)
);

alter table users
    owner to bdgcafe;

create index idx_users_name_trgm
    on users using gin (name gin_trgm_ops);

create index idx_users_email_trgm
    on users using gin (email gin_trgm_ops);
