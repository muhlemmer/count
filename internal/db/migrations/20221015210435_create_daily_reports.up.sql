create table count.daily_method_totals(
  day date not null default current_date,
  method_id int not null references count.methods(id),
  total int,

  primary key(day, method_id)
);
