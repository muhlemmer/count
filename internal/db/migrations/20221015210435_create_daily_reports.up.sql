create table count.daily_method_totals(
  day date not null default current_date,
  method_id bigint not null references count.methods(id),
  total bigint,

  primary key(day, method_id)
);
