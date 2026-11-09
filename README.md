# rd

Download reels and shorts to your machine from a share link.

```sh
rd https://www.facebook.com/share/r/<code>/
```

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/alileza/rd/main/install.sh | sh
```

This installs the latest release for macOS or Linux (Intel or ARM) into `/usr/local/bin`. It asks for `sudo` only if that directory isn't writable.

To install somewhere else, or to pin a version:

```sh
curl -fsSL https://raw.githubusercontent.com/alileza/rd/main/install.sh | RD_INSTALL_DIR="$HOME/.local/bin" sh
curl -fsSL https://raw.githubusercontent.com/alileza/rd/main/install.sh | RD_VERSION=v0.1.0 sh
```

With Go installed, you can also build from source:

```sh
go install github.com/alileza/rd/cmd/rd@latest
```

## Usage

```sh
rd <url>                  # save to the current directory as <site>-<id>.mp4
rd -o ~/Downloads <url>   # save into a directory
rd -o clip.mp4 <url>      # save to a specific file
rd -url <url>             # print the direct media URL without downloading
rd -version
```

## Supported sites

| Site      | Links                                                                 |
| --------- | --------------------------------------------------------------------- |
| Facebook  | `facebook.com/share/r/…`, `/reel/…`, `/videos/…`, `watch?v=…`, `fb.watch/…` |

Only public posts work. Facebook serves the best progressive MP4 it offers to logged-out users, usually 720p.

Instagram and YouTube Shorts are next.

## Adding a site

Each site is one file in [`extractor/`](extractor). It implements `Extractor` and registers itself in `init`:

```go
type Extractor interface {
	Name() string
	Match(u *url.URL) bool
	Extract(ctx context.Context, c *http.Client, u *url.URL) (*Media, error)
}
```

See [`extractor/facebook.go`](extractor/facebook.go) for an example.

## Releasing

Pushing a `v*` tag builds the binaries and publishes a GitHub release that the installer picks up:

```sh
git tag v0.1.0 && git push origin v0.1.0
```
