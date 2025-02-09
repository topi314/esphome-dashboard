FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS build

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

FROM alpine

RUN apk add --no-cache  \
    wkhtmltopdf \
    ttf-freefont

COPY --from=build /build/esphome-dashboard /bin/esphome-dashboard

EXPOSE 8080

ENTRYPOINT ["/bin/esphome-dashboard"]

CMD ["-config", "/var/lib/esphome-dashboard/config.toml"]