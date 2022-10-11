create schema count;

create table count.methods(
  id serial primary key,
  method varchar not null,
  path varchar not null,

  unique(method, path)
);

create table count.requests(
  method_id int not null references count.methods(id),
  request_timestamp timestamptz not null
);
