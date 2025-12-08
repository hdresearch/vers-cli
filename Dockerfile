FROM alpine:3.22.2

RUN apk add go musl-dev

WORKDIR /src

ADD . .

RUN CGO_ENABLED=1 CC=gcc \
    go build -ldflags="-linkmode external -extldflags '-static'" \
    -o bin/vers ./cmd/vers