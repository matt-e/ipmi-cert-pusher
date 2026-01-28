FROM golang:1.23-bookworm AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go build -o /ipmi-cert-pusher .

ARG SAA_URL=https://www.supermicro.com/wdl/utility/SUM/sum_2.14.0_Linux_x86_64_20240814.tar.gz
FROM debian:bookworm-slim AS saa-extract
ARG SAA_URL
RUN apt-get update && apt-get install -y --no-install-recommends curl ca-certificates && rm -rf /var/lib/apt/lists/*
RUN curl -fsSL "${SAA_URL}" -o /tmp/saa.tar.gz \
    && mkdir -p /opt/saa \
    && tar -xzf /tmp/saa.tar.gz -C /opt/saa --strip-components=1 \
    && rm /tmp/saa.tar.gz

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates libstdc++6 && rm -rf /var/lib/apt/lists/*
COPY --from=builder /ipmi-cert-pusher /usr/local/bin/ipmi-cert-pusher
COPY --from=saa-extract /opt/saa /opt/saa
ENTRYPOINT ["ipmi-cert-pusher"]
