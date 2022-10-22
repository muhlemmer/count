create schema count;

create table count.methods(
  id bigserial primary key,
  method varchar not null,
  path varchar not null,

  unique(method, path)
);

create table count.requests(
  method_id bigint not null references count.methods(id),
  request_timestamp timestamptz not null
);
