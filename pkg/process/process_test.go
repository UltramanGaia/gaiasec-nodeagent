package process

import "testing"

func TestSanitizeProcessStringRemovesInvalidUTF8(t *testing.T) {
	input := string([]byte{'n', 'g', 0xff, 'i', 'n', 'x'})

	got := sanitizeProcessString(input, "fallback")

	if got != "nginx" {
		t.Fatalf("sanitizeProcessString() = %q, want %q", got, "nginx")
	}
}

func TestSanitizeProcessStringFallsBackWhenEmptyAfterCleanup(t *testing.T) {
	input := string([]byte{0xff, 0xfe, ' '})

	got := sanitizeProcessString(input, "[unknown]")

	if got != "[unknown]" {
		t.Fatalf("sanitizeProcessString() = %q, want %q", got, "[unknown]")
	}
}
