FROM alpine:3.22.2

RUN apk add go

WORKDIR /src

ADD . .

RUN go build -o bin/vers ./cmd/vers