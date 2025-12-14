FROM golang:1.22-bookworm AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app .

FROM debian:12-slim

RUN apt-get update && apt-get install -y \
    sudo e2fsprogs util-linux pciutils procps coreutils bash lvm2 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/app .

ENTRYPOINT ["./app"]

