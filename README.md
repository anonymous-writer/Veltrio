# Veltrio

Veltrio is a lightweight, self-hosted download manager for Linux, built with Go and a responsive web interface.

## Features

- HTTP and HTTPS downloads
- Add, pause, resume, and delete downloads
- Live download progress, file size, and transfer speed
- Persistent download state
- Resume interrupted downloads after a restart when the remote server supports HTTP range requests
- Native Linux binaries for AMD64 and ARM64
- Multi-architecture Docker images for Linux AMD64 and ARM64
- YAML configuration with command-line overrides
- GitHub release and build status displayed in the web interface

> Current scope: Veltrio supports HTTP and HTTPS. FTP, BitTorrent, browser extensions, and parallel segmented downloads are not currently implemented.

## Quick start

Download the appropriate Linux binary or release archive from the [GitHub Releases](https://github.com/anonymous-writer/Veltrio/releases) page.

For example, on Linux AMD64:

```bash
chmod +x veltrio-linux-amd64
./veltrio-linux-amd64
```

On the first start, Veltrio creates a `config.yaml` file if one does not already exist. By default, data and download directories are located in the home directory of the user running Veltrio.

The web interface uses port **3010** by default. Open:

```text
http://localhost:3010
```

## Configuration

Example `config.yaml`:

```yaml
server:
  host: "0.0.0.0"
  port: 3010

paths:
  data: "~/.local/share/veltrio"
  downloads: "~/Downloads/Veltrio"
```

The `~` prefix is expanded to the home directory of the user running Veltrio. Environment variables such as `$HOME` are expanded as well.

## Command-line options

```bash
veltrio --help
veltrio --version
veltrio --config /path/to/config.yaml
veltrio --port 3010
veltrio --host 127.0.0.1
veltrio --data /path/to/data
veltrio --downloads /path/to/downloads
```

Explicit command-line options override environment variables and values from the YAML file.

Configuration precedence:

1. Command-line options
2. Environment variables
3. `config.yaml`

## Running with Docker

From the project directory:

```bash
docker compose up -d --build
```

Then open `http://localhost:3010`.

The Compose configuration stores downloads in `./Downloads` and application data (including the automatically created configuration file) in `./data`.

## Native builds

Requirements: Go as specified in `go.mod`.

Run tests and build the Linux AMD64 binary:

```bash
go test ./...
go vet ./...
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -o veltrio-linux-amd64 ./cmd/veltrio
```

Build the Linux ARM64 binary:

```bash
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -o veltrio-linux-arm64 ./cmd/veltrio
```

## Release archives

Pushing a version tag such as `v1.0.0` triggers the GitHub Actions release workflow:

```bash
git tag v1.0.0
git push origin v1.0.0
```

The workflow builds Linux AMD64 and ARM64 binaries and publishes release assets, including:

- `veltrio-linux-amd64.tar.gz`
- `veltrio-linux-arm64.tar.gz`
- Standalone binaries for both architectures
- `SHA256SUMS`
- The ARM64 update script and systemd service file

Download the ARM64 archive on a Raspberry Pi and follow the instructions in `scripts/update-veltrio-arm64.sh`.

## GitHub and build status

The web interface displays the latest published GitHub release and the timestamp of the most recent successful GitHub Actions run. The Veltrio host needs outbound internet access to reach the GitHub API for this information.

Repository: [github.com/anonymous-writer/Veltrio](https://github.com/anonymous-writer/Veltrio)

## License

Veltrio is distributed under the MIT License. See [LICENSE](LICENSE) for the full license text.

## Download settings

The `downloads` section controls parallel transfers, automatic resuming after startup, and whether deleting a download entry also removes its downloaded file. Set `delete_file_on_remove: false` to keep files on disk when removing entries. `logging.level` accepts `debug`, `info`, `warn`, or `error`.

```yaml
downloads:
  max_concurrent: 3
  resume_on_start: true
  delete_file_on_remove: true

logging:
  level: "info"
```


## Interface languages

Veltrio's web interface is available in German, English, and Spanish. Choose a language from the selector in the header; your choice is saved in the browser.
