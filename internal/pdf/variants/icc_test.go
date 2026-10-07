package variants

import "testing"

func TestSRGBProfileHeader(t *testing.T) {
	if len(SRGBProfile) < 128 {
		t.Fatalf("profile too short: %d bytes", len(SRGBProfile))
	}
	// ICC.1 §7.2.9: the profile file signature is "acsp" at offset 36.
	if got := string(SRGBProfile[36:40]); got != "acsp" {
		t.Errorf("signature = %q, want %q", got, "acsp")
	}
}
