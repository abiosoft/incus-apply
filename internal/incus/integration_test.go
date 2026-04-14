package incus

import (
	"os"
	"testing"
	"time"
)

func TestPingIntegration(t *testing.T) {
	if os.Getenv("INCUS_APPLY_INTEGRATION") == "" {
		t.Skip("set INCUS_APPLY_INTEGRATION=1 to run live Incus integration tests")
	}

	client := New(nil, "", false, false, 30*time.Second)
	if err := client.Ping(); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
}
