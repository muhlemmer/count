with result as (
    insert into count.methods (method, path)
        values ($1, $2)
        on conflict (method, path) do nothing
        returning id
)
insert into count.requests (method_id, request_timestamp)
    select id, $3 from result;
