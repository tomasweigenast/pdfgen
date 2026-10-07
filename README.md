# pdfgen

HTML/CSS to PDF rendering engine written in pure Go.

pdfgen lays out HTML and CSS for paged media and writes PDF files. It targets print: page sizes and margins, margin boxes, running headers and footers, counters, footnotes, cross references, bookmarks, and PDF/A and PDF/UA output.

Inspired by [WeasyPrint](https://github.com/Kozea/WeasyPrint).

> **Status:** early development. Nothing usable yet.

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

## Usage

Coming soon.

## License

See [LICENSE](LICENSE) and [NOTICE](NOTICE).
