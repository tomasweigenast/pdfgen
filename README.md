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

Requires Go 1.27 or later.

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
