package props

import (
	"os/exec"
	"testing"
)

func TestGeneratedFilesAreUpToDate(t *testing.T) {
	if testing.Short() {
		t.Skip("runs the generator")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go command not available")
	}
	out, err := exec.Command("go", "run", "./gen", "-check").CombinedOutput()
	if err != nil {
		t.Fatalf("generated files are stale, run go generate ./internal/css/props: %v\n%s", err, out)
	}
}
