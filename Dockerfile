FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS build
ARG TARGETOS=linux
ARG TARGETARCH
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/veltrio ./cmd/veltrio

FROM alpine:3.21
RUN addgroup -S veltrio && adduser -S -G veltrio veltrio && mkdir -p /app/data /app/Downloads && chown -R veltrio:veltrio /app
WORKDIR /app
COPY --from=build /out/veltrio /usr/local/bin/veltrio
USER veltrio
ENV VELTRIO_CONFIG=/app/data/config.yaml VELTRIO_ADDR=:9999 VELTRIO_DATA=/app/data VELTRIO_DOWNLOADS=/app/Downloads
EXPOSE 9999
VOLUME ["/app/data","/app/Downloads"]
ENTRYPOINT ["/usr/local/bin/veltrio"]
