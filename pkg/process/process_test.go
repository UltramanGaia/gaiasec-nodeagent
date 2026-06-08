package process

import (
	"testing"
	"time"
)

func TestCollectWithTimeoutReturnsResult(t *testing.T) {
	result, ok := collectWithTimeout(200*time.Millisecond, func() int {
		return 42
	})
	if !ok {
		t.Fatalf("expected result before timeout")
	}
	if result != 42 {
		t.Fatalf("expected 42, got %d", result)
	}
}

func TestCollectWithTimeoutTimesOut(t *testing.T) {
	start := time.Now()
	result, ok := collectWithTimeout(20*time.Millisecond, func() int {
		time.Sleep(100 * time.Millisecond)
		return 42
	})
	if ok {
		t.Fatalf("expected timeout")
	}
	if result != 0 {
		t.Fatalf("expected zero value on timeout, got %d", result)
	}
	if elapsed := time.Since(start); elapsed > 80*time.Millisecond {
		t.Fatalf("timeout returned too late: %s", elapsed)
	}
}
