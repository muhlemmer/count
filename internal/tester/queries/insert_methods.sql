insert into count.methods
    (method, path)
values
    ($1, $2)
returning id;
