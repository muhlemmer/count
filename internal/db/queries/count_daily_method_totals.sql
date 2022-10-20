with deleted as (
    delete from count.requests
    where request_timestamp::date = $1::date
    returning method_id
), inserted as (
    insert into count.daily_method_totals (day, method_id, total)
        select $1::date, method_id, count(*)
        from deleted
        group by method_id
    returning method_id, total
)
select method, path, total
from inserted
left join count.methods
on methods.id = inserted.method_id
