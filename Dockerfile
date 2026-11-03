FROM golang:1.25.4-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG APP_VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -tags nosqlite,web \
    -ldflags="-s -w -X github.com/GopeedLab/gopeed/pkg/base.Version=${APP_VERSION} -X github.com/GopeedLab/gopeed/pkg/base.InDocker=true" \
    -o /out/veltrio ./cmd/web

FROM alpine:3.23
RUN addgroup -S veltrio && adduser -S -G veltrio veltrio \
    && mkdir -p /app/data /app/Downloads \
    && chown -R veltrio:veltrio /app
WORKDIR /app
COPY --from=build /out/veltrio /usr/local/bin/veltrio
COPY config.json /app/config.json
VOLUME ["/app/data", "/app/Downloads"]
EXPOSE 9999
USER veltrio
ENTRYPOINT ["/usr/local/bin/veltrio", "--config", "/app/config.json"]
