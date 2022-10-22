select null, method, path, sum(total)::bigint
from count.daily_method_totals as dmt
join count.methods as m on m.id = dmt.method_id
where day
    between $1::date
    and $2::date
group by method, path
order by path, method;
