with deleted as (
    delete from count.requests
    where request_timestamp
        between $1
        and $2
    returning request_timestamp, method_id
), inserted as (
    insert into count.daily_method_totals (day, method_id, total)
        select request_timestamp::date, method_id, count(*)
        from deleted
        group by request_timestamp::date, method_id
    returning day, method_id, total
)
select day, method, path, total
from inserted
left join count.methods
on methods.id = inserted.method_id
order by day, path, method;
