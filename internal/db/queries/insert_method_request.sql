with result as (
    insert into count.methods (method, path)
        values ($1, $2)
        on conflict (method, path) do nothing
        returning id
)
insert into count.requests (method_id, request_timestamp)
    select id, $3::timestamptz
        from result
        where exists (select 1 from result)
    union all
    select id, $3::timestamptz
        from count.methods
        where method = $1
        and path = $2
        and not exists (select 1 from result);
