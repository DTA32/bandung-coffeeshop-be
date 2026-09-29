-- Weather suitability per cafe (BCF-1 "Match the weather").
-- Admin-defined list of the weather conditions a cafe suits; empty = never
-- matched by a weather filter. Values are constrained to the canonical set
-- in constants.Weather*; widen the CHECK when a new value is introduced.
alter table cafe
    add column weather text[] default '{}'::text[] not null
        constraint cafe_weather_valid
            check (weather <@ array ['clear', 'cloudy', 'rain']::text[]);

-- Serves the `weather && $n::text[]` overlap filter in /v1/search/cafes.
create index idx_cafe_weather
    on cafe using gin (weather);
