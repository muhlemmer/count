
FROM golang:1.19 as build

COPY . /build/
WORKDIR /build
RUN go build ./cmd/count

FROM busybox:glibc

COPY --from=build /etc/ssl/certs /etc/ssl/certs
COPY --from=build /build/count /usr/bin/count

ENV GRPC_LISTEN_ADDRESS=:7777
EXPOSE 7777

USER nobody
ENTRYPOINT [ "/usr/bin/count"]
