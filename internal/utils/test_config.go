package utils

import (
	"os"
	"testing"
)

func EnforceManualTest(t *testing.T) {
	if os.Getenv("RUN_MANUAL_TEST") != "1" {
		t.Skip("set RUN_MANUAL_TEST=1 to run manual LLM tests")
	}
}
