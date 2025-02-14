FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS build

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH \
    go build -o esphome-dashboard github.com/topi314/esphome-dashboard

FROM chromedp/headless-shell

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    fonts-freefont-ttf \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

COPY --from=build /build/esphome-dashboard /bin/esphome-dashboard

EXPOSE 8080

ENTRYPOINT ["/bin/esphome-dashboard"]

CMD ["-config", "/var/lib/esphome-dashboard/config.toml"]