// Package returns defines result objects shared by worker, sync, and mount flows.
package returns

import "testing"

// TestMountInfoFormatsNFSArguments verifies NFS host, source, options, and CLI rendering stay aligned.
func TestMountInfoFormatsNFSArguments(t *testing.T) {
	mount := MountInfo{Host: "storage.example", Path: "/exports/data/", Options: "ro,nolock"}

	if got, want := mount.Type(), "nfs"; got != want {
		t.Fatalf("Type() = %q, want %q", got, want)
	}
	if got, want := mount.Source(), "storage.example:/exports/data"; got != want {
		t.Fatalf("Source() = %q, want %q", got, want)
	}
	if got, want := mount.RefinedOptions(), "addr=storage.example,ro,nolock"; got != want {
		t.Fatalf("RefinedOptions() = %q, want %q", got, want)
	}
	wantArgs := []string{"-t", "nfs", "-o", "addr=storage.example,ro,nolock", "storage.example:/exports/data"}
	gotArgs := mount.MountArg()
	if len(gotArgs) != len(wantArgs) {
		t.Fatalf("MountArg() length = %d, want %d: %#v", len(gotArgs), len(wantArgs), gotArgs)
	}
	for i := range wantArgs {
		if gotArgs[i] != wantArgs[i] {
			t.Fatalf("MountArg()[%d] = %q, want %q", i, gotArgs[i], wantArgs[i])
		}
	}
	if got, want := mount.String(), "mount -t nfs -o addr=storage.example,ro,nolock storage.example:/exports/data"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

// TestMountInfoRefinedOptionsWithoutExtraOptions verifies addr=<host> is still emitted without custom options.
func TestMountInfoRefinedOptionsWithoutExtraOptions(t *testing.T) {
	mount := MountInfo{Host: "storage.example", Path: "volume"}
	if got, want := mount.RefinedOptions(), "addr=storage.example"; got != want {
		t.Fatalf("RefinedOptions() = %q, want %q", got, want)
	}
}
