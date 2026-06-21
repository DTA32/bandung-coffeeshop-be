CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS postgis;

create type location_type_enum as enum ('cafe', 'poi', 'area', 'district');

alter type location_type_enum owner to bdgcafe;

create type location_status_enum as enum ('active', 'closed', 'deleted');

alter type location_status_enum owner to bdgcafe;


create table location
(
    id               text                                                        not null
        primary key,
    name             text                                                        not null,
    description      text                 default ''::text                       not null,
    description_indo text                 default ''::text                       not null,
    gmaps_id         text,
    type             location_type_enum                                          not null,
    status           location_status_enum default 'active'::location_status_enum not null,
    coordinates      geometry(Geometry, 4326),
    created_at       timestamp            default now()                          not null,
    updated_at       timestamp            default now()                          not null
);

alter table location
    owner to bdgcafe;

create index idx_location_name_trgm
    on location using gin (name gin_trgm_ops);

create index idx_location_coordinates_gist
    on location using gist (coordinates);

create index idx_location_area_coordinates_gist
    on location using gist (coordinates)
    where ((type = 'area'::location_type_enum) AND (status = 'active'::location_status_enum));

create index idx_location_district_coordinates_gist
    on location using gist (coordinates)
    where ((type = 'district'::location_type_enum) AND (status = 'active'::location_status_enum));

create table location_image
(
    id               serial
        primary key,
    location_id      text                       not null
        references location,
    url              text                       not null,
    description      text      default ''::text not null,
    description_indo text      default ''::text not null,
    display_order    integer   default 0        not null,
    created_at       timestamp default now()    not null,
    updated_at       timestamp default now()    not null
);

alter table location_image
    owner to bdgcafe

create index idx_location_image_thumbnail
    on location_image (location_id, display_order, id);

create table cafe
(
    id          serial
        primary key,
    location_id text                    not null
        unique
        references location,
    instagram   text,
    open_hour   time,
    close_hour  time,
    is_featured boolean   default false not null,
    created_at  timestamp default now() not null,
    updated_at  timestamp default now() not null
);

alter table cafe
    owner to bdgcafe

create index idx_cafe_featured_updated
    on cafe (is_featured asc, updated_at desc);

create table cafe_review
(
    id             serial
        primary key,
    cafe_id        integer                 not null
        references cafe,
    is_subjective  boolean   default false not null,
    overall_score  numeric(3, 2),
    wfc_score      numeric(3, 2),
    personal_score numeric(3, 2),
    content        text,
    content_indo   text,
    visited_at     date,
    created_at     timestamp default now() not null,
    updated_at     timestamp default now() not null
);

alter table cafe_review
    owner to bdgcafe

create index idx_cafe_review_cafe_latest
    on cafe_review (cafe_id asc, visited_at desc nulls last, id desc);

create table tag
(
    id               serial
        primary key,
    name             text                       not null,
    name_indo        text      default ''::text not null,
    description      text      default ''::text not null,
    description_indo text      default ''::text not null,
    slug             text,
    created_at       timestamp default now()    not null,
    updated_at       timestamp default now()    not null
);

alter table tag
    owner to bdgcafe

create unique index idx_tag_slug
    on tag (slug)
    where (slug IS NOT NULL);

create index idx_tag_name_trgm
    on tag using gin (name gin_trgm_ops);

create index idx_tag_name_indo_trgm
    on tag using gin (name_indo gin_trgm_ops);

create table cafe_tag
(
    cafe_id    integer                 not null
        references cafe,
    tag_id     integer                 not null
        references tag,
    visible    boolean   default true  not null,
    created_at timestamp default now() not null,
    updated_at timestamp default now() not null,
    primary key (cafe_id, tag_id)
);

alter table cafe_tag
    owner to bdgcafe

create index idx_cafe_tag_tag_id
    on cafe_tag (tag_id, cafe_id)
    where visible;

create table cafe_price
(
    id               serial
        primary key,
    cafe_id          integer                 not null
        unique
        references cafe,
    price_range_min  integer,
    price_range_max  integer,
    coffee_price_min integer,
    coffee_price_max integer,
    snack_price_min  integer,
    snack_price_max  integer,
    food_price_min   integer,
    food_price_max   integer,
    created_at       timestamp default now() not null,
    updated_at       timestamp default now() not null
);

alter table cafe_price
    owner to bdgcafe

create index idx_cafe_price_min
    on cafe_price (price_range_min);

create index idx_cafe_price_max
    on cafe_price (price_range_max);

create table rating_type_label
(
    type       text                  not null
        primary key,
    label      text                  not null,
    label_indo text default ''::text not null
);

alter table rating_type_label
    owner to bdgcafe

create table rating_category
(
    id                     serial
        primary key,
    name                   text                       not null,
    name_indo              text      default ''::text not null,
    short_description      text      default ''::text not null,
    short_description_indo text      default ''::text not null,
    long_description       text      default ''::text not null,
    long_description_indo  text      default ''::text not null,
    slug                   text,
    type                   text                       not null
        references rating_type_label,
    lower_bound            numeric(10, 2)             not null,
    upper_bound            numeric(10, 2)             not null,
    created_at             timestamp default now()    not null,
    updated_at             timestamp default now()    not null
);

alter table rating_category
    owner to bdgcafe

create unique index idx_rating_category_type_slug
    on rating_category (type, slug)
    where (slug IS NOT NULL);

create index idx_rating_category_type_lb
    on rating_category (type, lower_bound);

create index idx_rating_category_name_trgm
    on rating_category using gin (name gin_trgm_ops);

create index idx_rating_category_name_indo_trgm
    on rating_category using gin (name_indo gin_trgm_ops);

create table cafe_rating
(
    id                              serial
        primary key,
    cafe_id                         integer                 not null
        references cafe,
    category_type                   text                    not null
        references rating_type_label,
    score                           numeric(10, 2)          not null,
    short_description_override      text,
    short_description_override_indo text,
    created_at                      timestamp default now() not null,
    updated_at                      timestamp default now() not null,
    unique (cafe_id, category_type)
);

alter table cafe_rating
    owner to bdgcafe

