package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// binary is the compiled command, built once for all the tests.
var binary string

func TestMain(m *testing.M) {
	os.Exit(buildAndRun(m))
}

func buildAndRun(m *testing.M) int {
	dir, err := os.MkdirTemp("", "pdfgen-cmd-test")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer os.RemoveAll(dir)

	binary = filepath.Join(dir, "pdfgen")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	if out, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build the command: %v\n%s", err, out)
		return 1
	}
	return m.Run()
}

type result struct {
	stdout, stderr []byte
	exit           int
}

// runCmd runs the command with the given standard input.
func runCmd(t *testing.T, stdin string, args ...string) result {
	t.Helper()
	cmd := exec.Command(binary, args...)
	cmd.Stdin = strings.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	res := result{stdout: stdout.Bytes(), stderr: stderr.Bytes()}
	if exitErr, ok := err.(*exec.ExitError); ok {
		res.exit = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("run %s: %v", binary, err)
	}
	return res
}

func parsePDF(t *testing.T, data []byte) *model.Context {
	t.Helper()
	api.DisableConfigDir()
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed
	ctx, err := api.ReadValidateAndOptimize(bytes.NewReader(data), conf)
	if err != nil {
		t.Fatalf("PDF parser rejects the output: %v", err)
	}
	return ctx
}

func readPDF(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func mediaBox(width, height string) []byte {
	return []byte(fmt.Sprintf("/MediaBox [0 0 %s %s]", width, height))
}

const page = `<!doctype html><title>Command test</title>
<style>@page { size: 80px 40px; margin: 0 }</style><div></div>`

func TestConvertsAFileToAFile(t *testing.T) {
	dir := t.TempDir()
	in := writeFile(t, dir, "in.html", page)
	out := filepath.Join(dir, "out.pdf")

	res := runCmd(t, "", in, out)
	if res.exit != 0 || len(res.stderr) != 0 {
		t.Fatalf("exit %d, stderr %q", res.exit, res.stderr)
	}
	if len(res.stdout) != 0 {
		t.Errorf("wrote %d bytes to standard output", len(res.stdout))
	}
	data := readPDF(t, out)
	ctx := parsePDF(t, data)
	if ctx.PageCount != 1 || ctx.Title != "Command test" {
		t.Errorf("page count = %d, title = %q", ctx.PageCount, ctx.Title)
	}
}

func TestConvertsRenderedText(t *testing.T) {
	dir := t.TempDir()
	in := writeFile(t, dir, "in.html", `<!doctype html><title>Text</title><h1>Hello</h1><p>Some paragraph of text.</p>`)
	out := filepath.Join(dir, "out.pdf")
	if res := runCmd(t, "", in, out); res.exit != 0 {
		t.Fatalf("exit %d, stderr %q", res.exit, res.stderr)
	}
	if got := parsePDF(t, readPDF(t, out)).PageCount; got != 1 {
		t.Errorf("page count = %d, want 1", got)
	}
}

func TestStandardInputAndOutput(t *testing.T) {
	res := runCmd(t, page, "-", "-", "--uncompressed-pdf")
	if res.exit != 0 || len(res.stderr) != 0 {
		t.Fatalf("exit %d, stderr %q", res.exit, res.stderr)
	}
	if got := parsePDF(t, res.stdout).Title; got != "Command test" {
		t.Errorf("title = %q", got)
	}
	if !bytes.Contains(res.stdout, mediaBox("60", "30")) {
		t.Errorf("PDF has no %s", mediaBox("60", "30"))
	}
}

func TestBaseURLOption(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "style.css", `@page { size: 60px 30px; margin: 0 }`)
	const doc = `<link rel="stylesheet" href="style.css">`

	res := runCmd(t, doc, "-u", dir+"/", "-", "-", "--uncompressed-pdf")
	if res.exit != 0 || len(res.stderr) != 0 {
		t.Fatalf("with a base URL: exit %d, stderr %q", res.exit, res.stderr)
	}
	if !bytes.Contains(res.stdout, mediaBox("45", "22.5")) {
		t.Errorf("the stylesheet was not applied: PDF has no %s", mediaBox("45", "22.5"))
	}

	res = runCmd(t, doc, "-", "-", "--uncompressed-pdf")
	if res.exit != 0 {
		t.Fatalf("without a base URL: exit %d, stderr %q", res.exit, res.stderr)
	}
	if bytes.Contains(res.stdout, mediaBox("45", "22.5")) {
		t.Error("a stylesheet was loaded without a base URL")
	}
	if !bytes.Contains(res.stderr, []byte("style.css")) {
		t.Errorf("stderr does not report the stylesheet: %q", res.stderr)
	}
}

func TestStylesheetOption(t *testing.T) {
	dir := t.TempDir()
	size := writeFile(t, dir, "size.css", `@page { size: 60px 30px }`)
	margin := writeFile(t, dir, "margin.css", `@page { margin: 0 }`)

	tests := []struct {
		name string
		args []string
	}{
		{"both given with -s", []string{"-s", size, "-s", margin}},
		{"long name and a URL", []string{"--stylesheet", "file://" + filepath.ToSlash(size), "--stylesheet=" + margin}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append(tt.args, "-", "-", "--uncompressed-pdf")
			res := runCmd(t, "<div>", args...)
			if res.exit != 0 || len(res.stderr) != 0 {
				t.Fatalf("exit %d, stderr %q", res.exit, res.stderr)
			}
			if !bytes.Contains(res.stdout, mediaBox("45", "22.5")) {
				t.Errorf("PDF has no %s", mediaBox("45", "22.5"))
			}
		})
	}
}

func TestMediaTypeOption(t *testing.T) {
	const doc = `<style media="screen">@page { size: 200px 200px; margin: 0 }</style>
		<style media="print">@page { size: 100px 100px; margin: 0 }</style>`
	tests := []struct {
		args []string
		box  []byte
	}{
		{nil, mediaBox("75", "75")},
		{[]string{"-m", "screen"}, mediaBox("150", "150")},
		{[]string{"--media-type", "print"}, mediaBox("75", "75")},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			res := runCmd(t, doc, append(tt.args, "-", "-", "--uncompressed-pdf")...)
			if res.exit != 0 {
				t.Fatalf("exit %d, stderr %q", res.exit, res.stderr)
			}
			if !bytes.Contains(res.stdout, tt.box) {
				t.Errorf("PDF has no %s", tt.box)
			}
		})
	}
}

func TestZoomOption(t *testing.T) {
	for _, arg := range [][]string{{"-z", "2"}, {"--zoom", "2"}, {"--zoom=2"}} {
		t.Run(strings.Join(arg, " "), func(t *testing.T) {
			res := runCmd(t, page, append(arg, "-", "-", "--uncompressed-pdf")...)
			if res.exit != 0 {
				t.Fatalf("exit %d, stderr %q", res.exit, res.stderr)
			}
			if !bytes.Contains(res.stdout, mediaBox("120", "60")) {
				t.Errorf("PDF has no %s", mediaBox("120", "60"))
			}
		})
	}
}

func TestUncompressedPDFOption(t *testing.T) {
	const doc = `<style>@page { size: 50px; margin: 0 }</style><div style="height: 10px; background: red"></div>`
	compressed := runCmd(t, doc, "-", "-")
	plain := runCmd(t, doc, "--uncompressed-pdf", "-", "-")
	if compressed.exit != 0 || plain.exit != 0 {
		t.Fatalf("exit %d and %d", compressed.exit, plain.exit)
	}
	if !bytes.Contains(compressed.stdout, []byte("/FlateDecode")) {
		t.Error("the default PDF is not compressed")
	}
	if bytes.Contains(plain.stdout, []byte("/FlateDecode")) {
		t.Error("the PDF made with --uncompressed-pdf has compressed streams")
	}
}

func TestOptionsMayFollowTheArguments(t *testing.T) {
	dir := t.TempDir()
	in := writeFile(t, dir, "in.html", page)
	out := filepath.Join(dir, "out.pdf")
	tests := []struct {
		name string
		args []string
	}{
		{"after", []string{in, out, "-z", "2", "--uncompressed-pdf"}},
		{"between", []string{in, "-z", "2", out, "--uncompressed-pdf"}},
		{"after a terminator", []string{"-z", "2", "--uncompressed-pdf", "--", in, out}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := runCmd(t, "", tt.args...); res.exit != 0 {
				t.Fatalf("exit %d, stderr %q", res.exit, res.stderr)
			}
			if data := readPDF(t, out); !bytes.Contains(data, mediaBox("120", "60")) {
				t.Errorf("PDF has no %s", mediaBox("120", "60"))
			}
			os.Remove(out)
		})
	}
}

func TestWarningsGoToStandardError(t *testing.T) {
	const doc = `<style>@page { size: 50px; margin: 0 }</style><img src="missing.png">`
	tests := []struct {
		name       string
		args       []string
		wantStderr []string
		noStderr   bool
	}{
		{"default", nil, []string{"Failed to load image"}, false},
		{"quiet", []string{"-q"}, nil, true},
		{"verbose", []string{"--verbose"}, []string{"Failed to load image", "Step 5"}, false},
	}
	dir := t.TempDir()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := runCmd(t, doc, append(tt.args, "-u", dir+"/", "-", "-")...)
			if res.exit != 0 {
				t.Fatalf("a warning changed the exit status to %d", res.exit)
			}
			parsePDF(t, res.stdout)
			if tt.noStderr && len(res.stderr) != 0 {
				t.Errorf("stderr = %q, want nothing", res.stderr)
			}
			for _, want := range tt.wantStderr {
				if !bytes.Contains(res.stderr, []byte(want)) {
					t.Errorf("stderr has no %q: %q", want, res.stderr)
				}
			}
		})
	}
}

func TestErrorsExitWithStatusOne(t *testing.T) {
	dir := t.TempDir()
	in := writeFile(t, dir, "in.html", page)
	missing := filepath.Join(dir, "missing.html")
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"no arguments", nil, "expected an input and an output"},
		{"no output", []string{in}, "expected an input and an output"},
		{"too many arguments", []string{in, "a.pdf", "b.pdf"}, "expected an input and an output"},
		{"unknown option", []string{"--no-such-option", in, "-"}, "no-such-option"},
		{"malformed zoom", []string{"-z", "big", in, "-"}, "flag -z"},
		{"negative zoom", []string{"-z", "-1", in, "-"}, "zoom"},
		{"verbose and quiet", []string{"-v", "-q", in, "-"}, "cannot be used together"},
		{"missing input", []string{missing, "-"}, "missing.html"},
		{"missing stylesheet", []string{"-s", filepath.Join(dir, "missing.css"), in, "-"}, "missing.css"},
		{"missing output directory", []string{in, filepath.Join(dir, "no", "such", "out.pdf")}, "out.pdf"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := runCmd(t, "", tt.args...)
			if res.exit != 1 {
				t.Errorf("exit status = %d, want 1", res.exit)
			}
			if !bytes.HasPrefix(res.stderr, []byte("pdfgen: ")) || !bytes.Contains(res.stderr, []byte(tt.want)) {
				t.Errorf("stderr = %q, want a message with %q", res.stderr, tt.want)
			}
			if len(res.stdout) != 0 {
				t.Errorf("wrote %d bytes to standard output", len(res.stdout))
			}
		})
	}
}

func TestFailedConversionLeavesNoOutputFile(t *testing.T) {
	dir := t.TempDir()
	in := writeFile(t, dir, "in.html", page)
	out := filepath.Join(dir, "out.pdf")
	if res := runCmd(t, "", "-z", "-1", in, out); res.exit != 1 {
		t.Fatalf("exit status = %d, want 1", res.exit)
	}
	if _, err := os.Stat(out); err == nil {
		t.Error("a failed conversion left an output file")
	}
}

func TestVersionAndHelp(t *testing.T) {
	res := runCmd(t, "", "--version")
	if res.exit != 0 || !bytes.HasPrefix(res.stdout, []byte("pdfgen ")) {
		t.Errorf("--version: exit %d, stdout %q", res.exit, res.stdout)
	}
	for _, flag := range []string{"-h", "--help"} {
		res = runCmd(t, "", flag)
		if res.exit != 0 || !bytes.HasPrefix(res.stdout, []byte("usage: pdfgen")) {
			t.Errorf("%s: exit %d, stdout %q", flag, res.exit, res.stdout)
		}
	}
}
