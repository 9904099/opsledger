FROM golang:1.22-bookworm AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /out/opsledger ./cmd/opsledger

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates openssh-client \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -r -u 10001 -m -d /var/lib/opsledger opsledger \
    && mkdir -p /data \
    && chown opsledger:opsledger /data

COPY --from=build /out/opsledger /usr/local/bin/opsledger

USER opsledger
WORKDIR /var/lib/opsledger
ENV OPSLEDGER_ADDR=0.0.0.0:18090
ENV OPSLEDGER_DATA=/data/opsledger.db
EXPOSE 18090
VOLUME ["/data"]

ENTRYPOINT ["opsledger"]
