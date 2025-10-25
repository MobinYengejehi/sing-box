FROM --platform=$BUILDPLATFORM golang:1.23.3 AS builder

RUN apt-get update && apt-get install -y git make build-essential ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /build

RUN git clone -b impl-wsc-protocol https://github.com/MobinYengejehi/sing-box.git .
RUN go mod download

RUN make build-libbox-static
RUN make build

FROM scratch AS artifacts

COPY --from=builder /build/bin/sing-box /sing-box
COPY --from=builder /build/bin/sing-box-libs /sing-box-libs
