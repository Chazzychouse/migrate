package dktesting_test

import (
	"regexp"
	"testing"
)

import (
	"github.com/golang-migrate/migrate/v4/dktesting"
)

func TestKernelVersion(t *testing.T) {
	release, err := dktesting.KernelVersion()
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^\d+\.\d+\.\d+`).MatchString(release) {
		t.Errorf("KernelVersion() = %q, want a kernel release like 6.8.0-1070-azure", release)
	}
}
