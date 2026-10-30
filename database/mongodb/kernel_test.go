package mongodb

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang-migrate/migrate/v4/dktesting"
	"github.com/moby/moby/client"
)

func TestMongo8RefusesKernel(t *testing.T) {
	cases := []struct {
		release string
		refuses bool
	}{
		{"6.8.0-1070-azure", false},
		{"6.17.0-1022-azure", false},
		{"6.19.0", true},
		{"6.19.8-arch1-1", true},
		{"7.0.0-1017-azure", true},
		{"7.0.13", true},
		{"7.0.14", false},
		{"7.1.0", false},
		{"7.2.8-9-omarchy", false},
		{"", false},
		{"not-a-kernel", false},
	}
	for _, c := range cases {
		if got := mongo8RefusesKernel(c.release); got != c.refuses {
			t.Errorf("mongo8RefusesKernel(%q) = %v, want %v", c.release, got, c.refuses)
		}
	}
}

// mongo8RefusesKernel reports whether MongoDB 8.0 refuses to start on a host whose
// kernel release string (as uname -r prints it) is release. mongod exits at startup
// on 6.19 up to, but not including, 7.0.14:
// https://jira.mongodb.org/browse/SERVER-121912
// https://jira.mongodb.org/browse/SERVER-125742
func mongo8RefusesKernel(release string) bool {
	var major, minor, patch int
	if _, err := fmt.Sscanf(release, "%d.%d.%d", &major, &minor, &patch); err != nil {
		return false
	}
	v := [3]int{major, minor, patch}
	return !less(v, [3]int{6, 19, 0}) && less(v, [3]int{7, 0, 14})
}

func less(a, b [3]int) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

func TestSpecsForKernel(t *testing.T) {
	all := []dktesting.ContainerSpec{
		{ImageName: "mongo:7.0"},
		{ImageName: "mongo:8.0"},
	}
	names := func(specs []dktesting.ContainerSpec) []string {
		out := make([]string, 0, len(specs))
		for _, s := range specs {
			out = append(out, s.ImageName)
		}
		return out
	}

	got, skipped := specsForKernel(all, "7.0.0-1017-azure")
	if fmt.Sprint(names(got)) != "[mongo:7.0]" || fmt.Sprint(skipped) != "[mongo:8.0]" {
		t.Errorf("refusing kernel: kept %v, skipped %v; want kept [mongo:7.0], skipped [mongo:8.0]", names(got), skipped)
	}

	got, skipped = specsForKernel(all, "6.8.0-1070-azure")
	if fmt.Sprint(names(got)) != "[mongo:7.0 mongo:8.0]" || len(skipped) != 0 {
		t.Errorf("unaffected kernel: kept %v, skipped %v; want all kept", names(got), skipped)
	}
}

// specsForKernel drops the MongoDB 8.0 image when the Docker host's kernel is one
// mongod refuses to start on, and returns the names of the images it dropped.
func specsForKernel(specs []dktesting.ContainerSpec, release string) ([]dktesting.ContainerSpec, []string) {
	if !mongo8RefusesKernel(release) {
		return specs, nil
	}
	var kept []dktesting.ContainerSpec
	var skipped []string
	for _, spec := range specs {
		if spec.ImageName == mongo8Image {
			skipped = append(skipped, spec.ImageName)
			continue
		}
		kept = append(kept, spec)
	}
	return kept, skipped
}

// dockerKernelVersion returns the kernel release of the Docker daemon's host, which is
// the kernel the test containers run on. It can differ from the kernel running the
// tests, e.g. with Docker Desktop or a remote Docker host.
func dockerKernelVersion(ctx context.Context) (release string, retErr error) {
	dc, err := client.New(client.FromEnv, client.WithAPIVersion("1.41"))
	if err != nil {
		return "", err
	}
	defer func() {
		if err := dc.Close(); err != nil && retErr == nil {
			retErr = fmt.Errorf("error closing Docker client: %w", err)
		}
	}()
	info, err := dc.Info(ctx, client.InfoOptions{})
	if err != nil {
		return "", err
	}
	return info.Info.KernelVersion, nil
}
