FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS build
ARG TARGETOS=linux
ARG TARGETARCH
ARG APP_VERSION=dev
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w -X main.version=${APP_VERSION}" -o /out/veltrio ./cmd/veltrio

FROM alpine:3.24
RUN addgroup -S veltrio && adduser -S -G veltrio veltrio && mkdir -p /app/data /app/Downloads && chown -R veltrio:veltrio /app
WORKDIR /app
COPY --from=build /out/veltrio /usr/local/bin/veltrio
USER veltrio
ENV VELTRIO_CONFIG=/app/data/config.yaml VELTRIO_ADDR=:3010 VELTRIO_DATA=/app/data VELTRIO_DOWNLOADS=/app/Downloads
EXPOSE 3010
VOLUME ["/app/data","/app/Downloads"]
ENTRYPOINT ["/usr/local/bin/veltrio"]
