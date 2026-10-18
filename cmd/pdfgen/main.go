// Command pdfgen renders an HTML document to PDF.
//
//	pdfgen [options] input output
//
// The input is a file path or a URL, or "-" for standard input. The output
// is a file path, or "-" for standard output. Warnings about content that
// could not be rendered go to standard error; they do not change the exit
// status, which is 0 on success and 1 on any error.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/tomasweigenast/pdfgen"
	"github.com/tomasweigenast/pdfgen/internal/logging"
)

const usageText = `usage: pdfgen [options] input output

Render an HTML document to PDF. The input is a file path or a URL, or - for
standard input. The output is a file path, or - for standard output.

options:
  -s, --stylesheet FILE    apply a user stylesheet, a file path or a URL;
                           can be repeated
  -u, --base-url URL       base URL that relative references resolve against
  -m, --media-type TYPE    media type of media queries (default "print")
  -z, --zoom FACTOR        scale the pages of the PDF by this factor
      --uncompressed-pdf   write the PDF without compressing it
  -v, --verbose            also report the steps of the rendering
  -q, --quiet              do not report warnings
      --version            print the version and exit
  -h, --help               print this help and exit
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// config holds what the command line asks for.
type config struct {
	input, output   string
	stylesheets     []string
	baseURL         string
	mediaType       string
	zoom            float64
	uncompressedPDF bool
	verbose, quiet  bool
	version         bool
}

// run executes the command and returns its exit status.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	cfg, err := parseArgs(args)
	switch {
	case errors.Is(err, flag.ErrHelp):
		fmt.Fprint(stdout, usageText)
		return 0
	case err != nil:
		fmt.Fprintf(stderr, "pdfgen: %v\nTry 'pdfgen --help' for more information.\n", err)
		return 1
	case cfg.version:
		fmt.Fprintf(stdout, "pdfgen %s\n", pdfgen.Version())
		return 0
	}

	level := slog.LevelWarn
	switch {
	case cfg.quiet:
		level = logging.Silent
	case cfg.verbose:
		level = slog.LevelInfo
		defer logging.SetProgressOutput(stderr)()
	}
	defer logging.SetOutput(stderr, level)()

	if err := convert(cfg, stdin, stdout); err != nil {
		fmt.Fprintf(stderr, "pdfgen: %v\n", err)
		return 1
	}
	return 0
}

// parseArgs reads the command line. Options may come before, between and
// after the two positional arguments.
func parseArgs(args []string) (*config, error) {
	cfg := &config{}
	fs := flag.NewFlagSet("pdfgen", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	addStylesheet := func(s string) error {
		cfg.stylesheets = append(cfg.stylesheets, s)
		return nil
	}
	fs.Func("s", "", addStylesheet)
	fs.Func("stylesheet", "", addStylesheet)
	for _, name := range []string{"u", "base-url"} {
		fs.StringVar(&cfg.baseURL, name, "", "")
	}
	for _, name := range []string{"m", "media-type"} {
		fs.StringVar(&cfg.mediaType, name, "", "")
	}
	for _, name := range []string{"z", "zoom"} {
		fs.Float64Var(&cfg.zoom, name, 0, "")
	}
	fs.BoolVar(&cfg.uncompressedPDF, "uncompressed-pdf", false, "")
	for _, name := range []string{"v", "verbose"} {
		fs.BoolVar(&cfg.verbose, name, false, "")
	}
	for _, name := range []string{"q", "quiet"} {
		fs.BoolVar(&cfg.quiet, name, false, "")
	}
	fs.BoolVar(&cfg.version, "version", false, "")

	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		rest := fs.Args()
		if consumed := len(args) - len(rest); consumed > 0 && args[consumed-1] == "--" {
			positional = append(positional, rest...)
			break
		}
		if len(rest) == 0 {
			break
		}
		positional = append(positional, rest[0])
		args = rest[1:]
	}

	if cfg.version {
		return cfg, nil
	}
	if len(positional) != 2 {
		return nil, fmt.Errorf("expected an input and an output, got %d arguments", len(positional))
	}
	if cfg.verbose && cfg.quiet {
		return nil, errors.New("--verbose and --quiet cannot be used together")
	}
	cfg.input, cfg.output = positional[0], positional[1]
	return cfg, nil
}

// convert renders the input and writes the PDF file.
func convert(cfg *config, stdin io.Reader, stdout io.Writer) error {
	var sourceOpts []pdfgen.SourceOption
	if cfg.baseURL != "" {
		sourceOpts = append(sourceOpts, pdfgen.WithBaseURL(cfg.baseURL))
	}
	if cfg.mediaType != "" {
		sourceOpts = append(sourceOpts, pdfgen.WithMediaType(cfg.mediaType))
	}

	var (
		doc *pdfgen.HTML
		err error
	)
	if cfg.input == "-" {
		doc, err = pdfgen.HTMLFromReader(stdin, sourceOpts...)
	} else {
		doc, err = pdfgen.HTMLGuess(cfg.input, sourceOpts...)
	}
	if err != nil {
		return err
	}

	opts := pdfgen.Options{Zoom: cfg.zoom, UncompressedPDF: cfg.uncompressedPDF}
	for _, s := range cfg.stylesheets {
		css, err := pdfgen.CSSGuess(s)
		if err != nil {
			return fmt.Errorf("stylesheet %s: %w", s, err)
		}
		opts.Stylesheets = append(opts.Stylesheets, css)
	}

	rendered, err := doc.Render(opts)
	if err != nil {
		return err
	}
	if cfg.output != "-" {
		return rendered.WriteFile(cfg.output, opts)
	}
	// A failure must not leave a truncated file on the pipe.
	var buf bytes.Buffer
	if err := rendered.WritePDF(&buf, opts); err != nil {
		return err
	}
	if _, err := buf.WriteTo(stdout); err != nil {
		return fmt.Errorf("write PDF: %w", err)
	}
	return nil
}
