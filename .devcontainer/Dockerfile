FROM golang:1.19

RUN apt-get update && \
    apt-get upgrade -y && \
    apt-get install -y \
        git ssh-client autoconf automake libtool curl make g++ unzip \
    && rm -rf /var/lib/apt/lists/*

ARG USER=vscode
ARG UID=1000
ARG GROUP=vscode
ARG GID=1000

RUN groupadd -g ${GID} ${GROUP}
RUN useradd -m -g ${GID} -s /bin/bash ${USER}

# VScode dev tools
USER ${USER}
RUN go install github.com/cweill/gotests/gotests@latest && \
    go install github.com/go-delve/delve/cmd/dlv@latest && \
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest && \
    go install github.com/mgechev/revive@latest && \
    go install github.com/ramya-rao-a/go-outline@latest && \
    go install github.com/uudashr/gopkgs/v2/cmd/gopkgs@latest && \
    go install golang.org/x/lint/golint@latest && \
    go install golang.org/x/tools/cmd/godoc@latest && \
    go install golang.org/x/tools/gopls@latest && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest && \
    go install honnef.co/go/tools/cmd/staticcheck@latest

# Protobuf compiler
ARG PROTOBUF_VERSION=3.21.7 JOBS=8
USER root
RUN git clone --branch v${PROTOBUF_VERSION} --depth 1 --recurse-submodules --shallow-submodules https://github.com/protocolbuffers/protobuf.git /protobuf && \
    cd /protobuf && \
    ./autogen.sh && \
    ./configure && \
    make -j${JOBS} && \
   # make -j${JOBS} check && \
    make install && ldconfig

USER ${USER}

# protobuf and gRPC Go plugins and tools
RUN go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest && \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# buf.build, connect.build tools
RUN go install github.com/bufbuild/buf/cmd/buf@latest && \
    go install github.com/bufbuild/connect-go/cmd/protoc-gen-connect-go@latest

# SQL migration tool
RUN go install -tags 'pgx,cockroachdb' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Cockroachlabs root cert
RUN curl --create-dirs -o /home/${USER}/.postgresql/root.crt -O https://cockroachlabs.cloud/clusters/9f955a92-b917-4eee-867b-c919cb7128fb/cert

ENV GRPC_LISTEN_ADDRESS=:7777

# godoc server and gRPC port
EXPOSE 6060 7777
