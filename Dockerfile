FROM golang:1.19 as build

COPY . /build/
WORKDIR /build
RUN go build ./cmd/count

FROM debian:stable-slim

ARG USER=count
ARG UID=1000
ARG GROUP=count
ARG GID=1000

RUN groupadd -g ${GID} ${GROUP}
RUN useradd -m -g ${GID} -s /bin/bash ${USER}

RUN apt-get update && \
    apt-get upgrade -y && \
    apt-get install -y curl && \
    rm -rf /var/lib/apt/lists/*

USER ${USER}
RUN curl --create-dirs -o /home/${USER}/.postgresql/root.crt -O https://cockroachlabs.cloud/clusters/9f955a92-b917-4eee-867b-c919cb7128fb/cert

USER root
COPY --from=build /build/count /usr/bin/count

ENV GRPC_LISTEN_ADDRESS=:7777
EXPOSE 7777

USER ${USER}
ENTRYPOINT ["/usr/bin/count"]
