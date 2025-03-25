create schema if not exists schema_name;

create table if not exists schema_name.orders
(
    id    bigserial not null
        constraint orders_pk
            primary key,
    name  text      not null,
    price integer default 0
);