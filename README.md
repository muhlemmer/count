# Count

[![Go](https://github.com/muhlemmer/count/actions/workflows/go.yml/badge.svg)](https://github.com/muhlemmer/count/actions/workflows/go.yml)
[![codecov](https://codecov.io/gh/muhlemmer/count/branch/main/graph/badge.svg?token=QUSPX5SMBH)](https://codecov.io/gh/muhlemmer/count)
[![Go Reference](https://pkg.go.dev/badge/github.com/muhlemmer/count.svg)](https://pkg.go.dev/github.com/muhlemmer/count)

Count is a request counting API, build for the Zitadel interview process.
It provides endpoints for request counting of other services
and retrieving the count metrics.

## Features

- PostgreSQL and CockroachDB support.
  The latter is confirmed by running tests Github actions against a cockroachdb free cloud offering.
- Counted "requests" are send over a streaming gRPC to the count API.
- High level queueing and server middleware are provided.
- Queues can be non-blocking and the API server uses concurrent jobs for database inserts.
  So a server posting to this API will not suffer from performance issues, even on connection
  failures to this API or between the API and database.

## Usage

### Input clients

Clients can dail a gRPC Client Connection to this API
and use [pkg/queue](https://pkg.go.dev/github.com/muhlemmer/count/pkg/queue)
to start sending request data:

```
cc, err := grpc.DialContext(context.TODO(), "count.muhlemmer.com:443",
    grpc.WithTransportCredentials(insecure.NewCredentials()),
    grpc.WithBlock(),
)
if err != nil {
    panic(err)
}

q, err := NewCountAddClient(context.TODO(), cc)
if err != nil {
    panic(err)
}

q.QueueOrDrop(context.TODO(), &countv1.AddRequest{
    Method:           countv1.Method_GET,
    Path:             "/foo/bar",
    RequestTimestamp: timestamppb.Now(),
})
```

HTTP servers can use [Middleware](https://pkg.go.dev/github.com/muhlemmer/count/pkg/queue#CountAddQueue.Middleware) instead:

```
s := &http.Server{
    Addr:    ":8080",
    Handler: q.Middleware(http.DefaultServeMux),
}
s.ListenAndServe()
```

Or [UnaryInterceptor](https://pkg.go.dev/github.com/muhlemmer/count/pkg/queue#CountAddQueue.UnaryInterceptor) for gRPC servers:

```
grpc.NewServer(grpc.ChainUnaryInterceptor(
    q.UnaryInterceptor(),
))
```

### Retrieval clients

Clients which want to retrieve metrics can use gRPC.
API documenation is available at https://buf.build/muhlemmer/count/docs/main:count.v1.

If this API where to be used in producion, I would consider moving to https://connect.build/ as gRPC and REST protocol, which for now has a too big impact.

### Server

Easiest way to run a count API server, is to use the
Docker image, which is build automatically by the CI/CD.
First, create a `.env` file:

```
# Driver name for the migrations.
# Use `pgx` for postgresql and
# `cockroachdb` for cockroachdb.
MIGRATION_DRIVER=cockroachdb

# Database connection URL, including secrets.
DB_URL=postgresql://<user>:<password>@<host>:<port>/<db>?sslmode=verify-full&options=--cluster%3D<your-cockroachdb-cluster>
```

Then, start the server with Docker:

```
docker run --env-file .env -p 7777:7777 ghcr.io/muhlemmer/count:main
```

## Architecture

The design goal of this project was to "increase a counter when a API request
is made", in PostgreSQL or CockroachDB. However keeping a table with API
method names and a incremental counter would not scale well. Requests can
happen concurrently and `UPDATE`s on the same row will need to be serialized by
the database. In the case of CockroachDB this would require distributed locks
with concensus through the raft log. Due to lock contention the count API would
not be able to keep up.

Instead, for every request a timestamp is inserted in a table, along with a `method_id` which is a foreign key to a method and path table. `INSERT` does not require locking and can happen on the same table on multiple nodes concurrenly. CockroachDB takes care of replicating and merging the records in the background.

Daily a cron job can call the
[CountDailyTotals](https://buf.build/muhlemmer/count/docs/main:count.v1#count.v1.CountService.CountDailyTotals) endpoint.
This counts the requests by `method_id`, stores the result in a
`daily_method_totals` table while deleting all rows for that day from the
`requests` table. This keeps storage size pretty decent. Both `int` and `timestamptz` take 8 bytes, so 16 bytes per row. 1 milion request counts per day would result in just 16MB of storage by the end of each day.

As such it is "cheap" to read periodic reports from the `daily_method_totals` table , such as yealy, monthly or daily.

## Lessons learned

Some hickups in the process where encountered. As CockroachDB is supposed to be
somewhat compatible with PostgreSQL, I initally felt confident to go the
CockroachDB way, even without previous experience. This cost me more time as
expected due to dirrent incompatibilties and suprises:

1. The cockroachdb Docker image installation process is interactive, unlike the
  PosgreSQL docker image which can be configured using environment variable
  This makes cockroachDB unsuitable for devcontainers.
2. cockroachdb doesn't have `pg_advisory_lock`, which is used by the migrations
  tool I know. Instead, I had to learn to use `golang-migrate` which came with 
  its own pitfalls. Besides, due to the fact a seperate lock table must be used 
  by the tool, any failure would lead in stale locks and require manual 
  cleanups. (probably a bug the tool, I did not invesigate).
3. Because of point 1, I opted to use a free serverless instance from cockroach 
  labs. Due to physical distance to the cloud location I experienced high 
  network latency >100ms which made it a pain to `INSERT` test data. As a 
  second option I tried to use `COPY FROM`, which bugged out on me. https://
  github.com/cockroachdb/cockroach/issues/52722
4. CockroachDB uses 8 bytes for all integer types, where PostgreSQL uses 4 
  bytes for `int`. When defining a `int` column and using a `pgtype.Int4` in 
  the code, errors would occur when connecting to CockroachDB later, due to 
  witdh mismatch. Even when the column is defined as `int`, cockroach still 
  sends `bigint` on the wire. Using just `pgtype.Int8` gave me trouble with 
  PostgreSQL. So in the end I defined all integer columns as `bigint` with 
  `pgtype.Int8` to solve the porability issue.

It was a bit of a learning curve and although I do see the positive case for
production use of CockroachDB, it is still a bit of a pain as development DB.
In this case I opted to develop against a local PostgreSQL instance first and
test against CockroachDB later and in CI.
