create extension "uuid-ossp";

create table requests
(
    id      uuid primary key default uuid_generate_v4(),
    method  varchar not null,
    schema  varchar not null,
    host    varchar not null,
    path    varchar not null,
    headers jsonb   not null,
    body    varchar not null
);

create index persons_id on requests (id);
