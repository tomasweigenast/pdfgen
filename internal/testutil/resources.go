package testutil

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ModuleRoot returns the directory holding go.mod, found by walking up from
// the working directory of the running test.
func ModuleRoot(t testing.TB) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found in any parent directory")
		}
		dir = parent
	}
}

// ResourcePath returns the absolute path of a file in testdata/resources.
func ResourcePath(t testing.TB, name string) string {
	t.Helper()
	return filepath.Join(ModuleRoot(t), "testdata", "resources", name)
}

// ResourceURL returns the file URL of a file in testdata/resources. The file
// does not have to exist, so a dummy name gives a base URL in the right
// directory.
func ResourceURL(t testing.TB, name string) string {
	t.Helper()
	p := filepath.ToSlash(ResourcePath(t, name))
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return (&url.URL{Scheme: "file", Path: p}).String()
}

// ResultsDir returns testdata/results, creating it if needed. Images of
// failed comparisons are written there.
func ResultsDir(t testing.TB) string {
	t.Helper()
	dir := filepath.Join(ModuleRoot(t), "testdata", "results")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create results directory: %v", err)
	}
	return dir
}

// ResultName derives a file name from the test name: every character that
// is not a letter or digit becomes an underscore.
func ResultName(t testing.TB) string {
	name := []rune(t.Name())
	for i, r := range name {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z') {
			name[i] = '_'
		}
	}
	return strings.Trim(string(name), "_")
}
