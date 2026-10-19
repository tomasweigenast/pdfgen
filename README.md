# pdfgen

HTML/CSS to PDF rendering engine written in pure Go.

pdfgen lays out HTML and CSS for paged media and writes PDF files. It targets print: page sizes and margins, margin boxes, running headers and footers, counters, footnotes, cross references, bookmarks, and PDF/A and PDF/UA output.

Inspired by [WeasyPrint](https://github.com/Kozea/WeasyPrint).

> **Status:** early development. Simple documents already render end to end; tables, flexbox, grid, multi-column and SVG are not implemented yet.

## Goals

- Pure Go, no cgo required.
- CSS 2.1 plus the modern modules that matter for print:
  - Paged Media and Generated Content for Paged Media;
  - Fragmentation;
  - Flexbox, Grid and Multi-column;
  - Fonts 4, Text 3 and Colors 4/5.
- Correct text shaping (ligatures, kerning, complex scripts) and hyphenation.
- Built-in SVG renderer.
- Tagged PDF, PDF/A, PDF/UA and PDF/X variants.
- Deterministic output.

## Quick start

Building from source requires Go 1.27 or later. Prebuilt binaries do not require Go.

### Downloads

Download archives and the checksum file from [GitHub Releases](https://github.com/tomasweigenast/pdfgen/releases).
Supported targets: macOS, Linux and Windows, each for AMD64 and ARM64. Extract
`pdfgen` (or `pdfgen.exe`) and put it on your PATH. Direct binaries use system or
document-provided fonts. Archives include third-party dictionary notices and
source data in `third-party/hyphenation`.

For example, verify an archive with the matching release checksum file:

```sh
# Linux (checks downloaded files and skips other targets):
sha256sum --ignore-missing -c pdfgen_0.1.0_checksums.txt
# macOS (compare the result with the matching line in the checksum file):
shasum -a 256 pdfgen_0.1.0_darwin_arm64.tar.gz
```

```powershell
Get-FileHash .\pdfgen_0.1.0_windows_amd64.zip -Algorithm SHA256
# Compare with that filename in pdfgen_0.1.0_checksums.txt.
```

### Docker

The image at `ghcr.io/tomasweigenast/pdfgen` supports Linux AMD64 and ARM64 and
includes DejaVu fonts and CA certificates. It runs as non-root. Give the selected
user write permission to the mounted output directory:

```sh
docker run --rm \
  --user "$(id -u):$(id -g)" \
  -v "$PWD:/work" \
  ghcr.io/tomasweigenast/pdfgen:0.1.0 input.html output.pdf

cat input.html | docker run --rm -i \
  ghcr.io/tomasweigenast/pdfgen:0.1.0 - - > output.pdf
```

Mount additional fonts read-only at `/usr/share/fonts/custom`, for example
`-v "$PWD/fonts:/usr/share/fonts/custom:ro"`. Exact version tags and digests can
be pinned; stable `latest`, minor and (from 1.0) major aliases are also available.
Release candidates only receive their exact version tag.

### Command line

Build the binary from a checkout:

```sh
go build -o pdfgen ./cmd/pdfgen
```

Convert a file, a URL or standard input:

```sh
pdfgen invoice.html invoice.pdf
pdfgen https://example.org/ example.pdf
cat invoice.html | pdfgen - - > invoice.pdf
```

Common options:

```sh
pdfgen -s extra.css invoice.html invoice.pdf     # add a user stylesheet (repeatable)
pdfgen -u https://example.org/ page.html out.pdf # base URL for relative links
pdfgen -z 1.5 invoice.html invoice.pdf           # scale the pages
pdfgen -m screen invoice.html invoice.pdf        # media type for @media rules
pdfgen --uncompressed-pdf invoice.html out.pdf   # readable PDF, for debugging
pdfgen -v invoice.html invoice.pdf               # show the rendering steps
pdfgen -q invoice.html invoice.pdf               # hide warnings
pdfgen --help
```

Warnings go to standard error. The exit status is 0 on success and 1 on any error.

### Go library

```go
package main

import (
	"log"

	"github.com/tomasweigenast/pdfgen"
)

func main() {
	doc, err := pdfgen.HTMLFromString(`
		<style>
			@page { size: A4; margin: 20mm }
			h1 { color: #1f4e79 }
		</style>
		<h1>Hello</h1>
		<p>Written by pdfgen.</p>`)
	if err != nil {
		log.Fatal(err)
	}
	if err := doc.WriteFile("hello.pdf", pdfgen.Options{}); err != nil {
		log.Fatal(err)
	}
}
```

Documents can also be loaded with `HTMLFromFile`, `HTMLFromURL` and `HTMLFromReader`. Options such as a base URL, a custom fetcher or the media type are passed when loading:

```go
doc, err := pdfgen.HTMLFromFile("page.html", pdfgen.WithBaseURL("https://example.org/"))
```

Extra stylesheets and PDF settings go in `Options`:

```go
err = doc.WriteFile("out.pdf", pdfgen.Options{
	Stylesheets: []*pdfgen.CSS{pdfgen.CSSFromString("body { font-size: 11pt }")},
	Zoom:        1.25,
})
```

To keep the laid out pages and write them more than once, call `Render` and then `WritePDF` or `WriteFile` on the returned `Document`.

## Development

```sh
go test ./...
go vet ./...
staticcheck ./...
```

Pixel tests rasterize with Ghostscript (`gs`) and skip when it is not installed.

## License

See [LICENSE](LICENSE) and [NOTICE](NOTICE).
