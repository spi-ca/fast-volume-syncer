//go:build integration && nfs && linux
// +build integration,nfs,linux

// Package main contains privileged Linux/NFS integration coverage for the fast-volume-syncer binary.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

const (
	fvsTestSrcNFSHost          = "FVS_TEST_SRC_NFS_HOST"
	fvsTestSrcNFSExport        = "FVS_TEST_SRC_NFS_EXPORT"
	fvsTestDstNFSHost          = "FVS_TEST_DST_NFS_HOST"
	fvsTestDstNFSExport        = "FVS_TEST_DST_NFS_EXPORT"
	fvsTestSrcSubPath          = "FVS_TEST_SRC_SUBPATH"
	fvsTestDstSubPath          = "FVS_TEST_DST_SUBPATH"
	fvsTestSrcVerifyRoot       = "FVS_TEST_SRC_VERIFY_ROOT"
	fvsTestDstVerifyRoot       = "FVS_TEST_DST_VERIFY_ROOT"
	defaultSrcMountOptions     = "ro,nodiratime,noatime,vers=3,rsize=524288,wsize=524288,hard,intr,nolock,proto=tcp,timeo=600,retrans=2,sec=sys"
	defaultDstMountOptions     = "rw,nodiratime,noatime,vers=3,rsize=524288,wsize=524288,hard,intr,nolock,proto=tcp,timeo=600,retrans=2,sec=sys"
	requiredCapabilitySysAdmin = uint(21)
)

// nfsSyncSandboxConfig carries the target-environment values needed by the opt-in NFS integration test.
type nfsSyncSandboxConfig struct {
	// SourceNFSHost is the NFS server used for the source mount.
	SourceNFSHost string
	// SourceNFSExport is the source export path passed through the selector CSV.
	SourceNFSExport string
	// DestinationNFSHost is the NFS server used for the destination mount.
	DestinationNFSHost string
	// DestinationNFSExport is the destination export path passed through the selector CSV.
	DestinationNFSExport string
	// SourceSubPath is the fixture subpath copied from the source export.
	SourceSubPath string
	// DestinationSubPath is the disposable subpath copied into the destination export.
	DestinationSubPath string
	// SourceVerifyRoot is a locally mounted or otherwise accessible root used to prepare source fixtures.
	SourceVerifyRoot string
	// DestinationVerifyRoot is a locally mounted or otherwise accessible root used to verify destination fixtures.
	DestinationVerifyRoot string
}

// TestNFSSyncSandboxE2E is a privileged skeleton for target-environment NFS sandbox checks.
func TestNFSSyncSandboxE2E(t *testing.T) {
	cfg := loadNFSSyncSandboxConfig(t)
	requireNFSSyncSandboxPrivileges(t)

	tmp := t.TempDir()
	binPath := filepath.Join(tmp, "fast-volume-syncer")
	build := exec.Command("go", "build", "-o", binPath, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build fast-volume-syncer: %v\n%s", err, out)
	}

	srcFixtureRoot := joinVerifiedSubpath(t, cfg.SourceVerifyRoot, cfg.SourceSubPath, fvsTestSrcVerifyRoot, fvsTestSrcSubPath)
	dstFixtureRoot := joinVerifiedSubpath(t, cfg.DestinationVerifyRoot, cfg.DestinationSubPath, fvsTestDstVerifyRoot, fvsTestDstSubPath)

	if err := os.RemoveAll(srcFixtureRoot); err != nil {
		t.Fatalf("clear source fixture root: %v", err)
	}
	if err := os.MkdirAll(srcFixtureRoot, 0o755); err != nil {
		t.Fatalf("prepare source fixture root: %v", err)
	}
	requireNoSymlinkPathComponents(t, filepath.Dir(srcFixtureRoot), filepath.Base(srcFixtureRoot), fvsTestSrcSubPath)

	payload := []byte("nfs sync sandbox e2e payload\n")
	sourceFilePath := filepath.Join(srcFixtureRoot, "nested", "file.txt")
	sourceLinkPath := filepath.Join(srcFixtureRoot, "link.txt")
	if err := os.MkdirAll(filepath.Dir(sourceFilePath), 0o755); err != nil {
		t.Fatalf("prepare source fixture directory: %v", err)
	}
	requireNoSymlinkPathComponents(t, filepath.Dir(srcFixtureRoot), filepath.Base(srcFixtureRoot), fvsTestSrcSubPath)
	if err := os.WriteFile(sourceFilePath, payload, 0o644); err != nil {
		t.Fatalf("write source fixture: %v", err)
	}
	if err := os.Remove(sourceLinkPath); err != nil && !os.IsNotExist(err) {
		t.Fatalf("clear source symlink: %v", err)
	}
	if err := os.Symlink("nested/file.txt", sourceLinkPath); err != nil {
		t.Fatalf("write source symlink: %v", err)
	}

	if err := os.RemoveAll(dstFixtureRoot); err != nil {
		t.Fatalf("clear destination fixture root: %v", err)
	}
	if err := os.MkdirAll(dstFixtureRoot, 0o755); err != nil {
		t.Fatalf("prepare destination fixture root: %v", err)
	}
	requireNoSymlinkPathComponents(t, filepath.Dir(dstFixtureRoot), filepath.Base(dstFixtureRoot), fvsTestDstSubPath)

	csvPath := filepath.Join(tmp, "one-row.csv")
	writeNFSSelectorCSV(t, csvPath, cfg)

	cmd := exec.Command(
		binPath,
		"select",
		"0",
		csvPath,
	)
	cmd.Env = []string{
		"PATH=/usr/sbin:/usr/bin:/sbin:/bin",
		"SRC_STORAGE_MOUNT_HOST=" + cfg.SourceNFSHost,
		"SRC_STORAGE_MOUNT_OPTION=" + defaultSrcMountOptions,
		"SRC_STORAGE_MOUNT_NAME=src",
		"DST_STORAGE_MOUNT_HOST=" + cfg.DestinationNFSHost,
		"DST_STORAGE_MOUNT_OPTION=" + defaultDstMountOptions,
		"DST_STORAGE_MOUNT_NAME=dst",
		"REPORT_ENABLED=true",
		"SANDBOX_DISABLED=false",
		"SCAN_FIND_PATH=",
		"RETRY_ATTEMPTS=0",
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run sync sandbox skeleton: %v\n%s", err, out)
	}
	requireSandboxRunEvidence(t, string(out))
	requireNoHostSyncerMountLeaks(t)

	destinationFilePath := filepath.Join(dstFixtureRoot, "nested", "file.txt")
	destinationBytes, err := os.ReadFile(destinationFilePath)
	if err != nil {
		t.Fatalf("read destination file: %v", err)
	}
	if !bytes.Equal(destinationBytes, payload) {
		t.Fatalf("destination payload = %q, want %q", destinationBytes, payload)
	}

	sourceChecksum, err := sha256File(sourceFilePath)
	if err != nil {
		t.Fatalf("checksum source file: %v", err)
	}
	destinationChecksum, err := sha256File(destinationFilePath)
	if err != nil {
		t.Fatalf("checksum destination file: %v", err)
	}
	if destinationChecksum != sourceChecksum {
		t.Fatalf("destination checksum = %s, want %s", destinationChecksum, sourceChecksum)
	}

	sourceLinkTarget, err := os.Readlink(sourceLinkPath)
	if err != nil {
		t.Fatalf("read source symlink: %v", err)
	}
	destinationLinkTarget, err := os.Readlink(filepath.Join(dstFixtureRoot, "link.txt"))
	if err != nil {
		t.Fatalf("read destination symlink: %v", err)
	}
	if destinationLinkTarget != sourceLinkTarget {
		t.Fatalf("destination symlink target = %q, want %q", destinationLinkTarget, sourceLinkTarget)
	}
}

// loadNFSSyncSandboxConfig reads the privileged NFS integration inputs from FVS_TEST_* environment variables.
func loadNFSSyncSandboxConfig(t *testing.T) nfsSyncSandboxConfig {
	t.Helper()

	values := map[string]string{
		fvsTestSrcNFSHost:    strings.TrimSpace(os.Getenv(fvsTestSrcNFSHost)),
		fvsTestSrcNFSExport:  strings.TrimSpace(os.Getenv(fvsTestSrcNFSExport)),
		fvsTestDstNFSHost:    strings.TrimSpace(os.Getenv(fvsTestDstNFSHost)),
		fvsTestDstNFSExport:  strings.TrimSpace(os.Getenv(fvsTestDstNFSExport)),
		fvsTestSrcSubPath:    strings.TrimSpace(os.Getenv(fvsTestSrcSubPath)),
		fvsTestDstSubPath:    strings.TrimSpace(os.Getenv(fvsTestDstSubPath)),
		fvsTestSrcVerifyRoot: strings.TrimSpace(os.Getenv(fvsTestSrcVerifyRoot)),
		fvsTestDstVerifyRoot: strings.TrimSpace(os.Getenv(fvsTestDstVerifyRoot)),
	}

	missing := make([]string, 0, len(values))
	for name, value := range values {
		if value == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		t.Skipf("set %s to run the privileged NFS sync sandbox skeleton", strings.Join(missing, ", "))
	}

	return nfsSyncSandboxConfig{
		SourceNFSHost:         values[fvsTestSrcNFSHost],
		SourceNFSExport:       values[fvsTestSrcNFSExport],
		DestinationNFSHost:    values[fvsTestDstNFSHost],
		DestinationNFSExport:  values[fvsTestDstNFSExport],
		SourceSubPath:         values[fvsTestSrcSubPath],
		DestinationSubPath:    values[fvsTestDstSubPath],
		SourceVerifyRoot:      values[fvsTestSrcVerifyRoot],
		DestinationVerifyRoot: values[fvsTestDstVerifyRoot],
	}
}

// requireNFSSyncSandboxPrivileges skips the test unless the process can perform mount operations.
// writeNFSSelectorCSV writes a one-row selector input so the test uses the real selector-launched sync child path.
func writeNFSSelectorCSV(t *testing.T, path string, cfg nfsSyncSandboxConfig) {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create selector csv: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	rows := [][]string{
		{
			"node", "source_volume", "destination_volume", "source_path", "destination_path",
			"source_project_id", "source_project_name", "used_size", "used_size_human",
			"volume_type", "volume_size", "volume_size_human", "destination_project_name",
			"volume_name", "source_volume_key",
		},
		{
			"0", cfg.SourceNFSExport, cfg.DestinationNFSExport, cfg.SourceSubPath, cfg.DestinationSubPath,
			"1", "nfs-source", "0", "0B", "nfs", "0", "0B", "nfs-destination",
			"test-volume", "dummy-non-secret-source-volume-key",
		},
	}
	if err := writer.WriteAll(rows); err != nil {
		t.Fatalf("write selector csv: %v", err)
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		t.Fatalf("flush selector csv: %v", err)
	}
}

// requireNFSSyncSandboxPrivileges skips the test unless the process can perform mount operations.
func requireNFSSyncSandboxPrivileges(t *testing.T) {
	t.Helper()

	hasCapSysAdmin, err := hasEffectiveCapability(requiredCapabilitySysAdmin)
	if err != nil {
		if os.Geteuid() != 0 {
			t.Skipf("need root or CAP_SYS_ADMIN for mount-based NFS integration: %v", err)
		}
		t.Skipf("could not confirm CAP_SYS_ADMIN for privileged NFS integration: %v", err)
	}
	if os.Geteuid() != 0 && !hasCapSysAdmin {
		t.Skip("need root or CAP_SYS_ADMIN for mount-based NFS integration")
	}
	if os.Geteuid() == 0 && !hasCapSysAdmin {
		t.Skip("root process lacks CAP_SYS_ADMIN; skipping mount-based NFS integration")
	}
}

// hasEffectiveCapability checks the Linux CapEff bitmask for a single capability number.
func hasEffectiveCapability(capability uint) (bool, error) {
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return false, fmt.Errorf("read /proc/self/status: %w", err)
	}
	const prefix = "CapEff:"
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		raw := strings.TrimSpace(strings.TrimPrefix(line, prefix))
		value, err := strconv.ParseUint(raw, 16, 64)
		if err != nil {
			return false, fmt.Errorf("parse CapEff %q: %w", raw, err)
		}
		return value&(uint64(1)<<capability) != 0, nil
	}
	return false, fmt.Errorf("CapEff not found in /proc/self/status")
}

// joinVerifiedSubpath ensures fixture setup stays within the caller-provided disposable root.
func joinVerifiedSubpath(t *testing.T, root, subpath, rootEnvName, subpathEnvName string) string {
	t.Helper()

	resolvedRoot := resolveVerifiedRoot(t, root, rootEnvName)
	cleanSubpath := filepath.Clean(subpath)
	if cleanSubpath == "." || cleanSubpath == string(os.PathSeparator) {
		t.Fatalf("%s=%q must point to a non-root disposable subpath", subpathEnvName, subpath)
	}
	if filepath.IsAbs(subpath) {
		t.Fatalf("%s=%q must be relative to %s", subpathEnvName, subpath, rootEnvName)
	}

	joined := filepath.Join(resolvedRoot, cleanSubpath)
	rel, err := filepath.Rel(resolvedRoot, joined)
	if err != nil {
		t.Fatalf("resolve %s/%s: %v", rootEnvName, subpathEnvName, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		t.Fatalf("%s=%q escapes %s=%q", subpathEnvName, subpath, rootEnvName, root)
	}

	requireNoSymlinkPathComponents(t, resolvedRoot, rel, subpathEnvName)
	return joined
}

// resolveVerifiedRoot canonicalizes a fixture root and rejects unsafe or shared locations.
func resolveVerifiedRoot(t *testing.T, root, rootEnvName string) string {
	t.Helper()

	cleanRoot := filepath.Clean(root)
	if cleanRoot == "." || cleanRoot == string(os.PathSeparator) {
		t.Fatalf("%s=%q is unsafe for destructive integration-test setup", rootEnvName, root)
	}

	absRoot, err := filepath.Abs(cleanRoot)
	if err != nil {
		t.Fatalf("resolve absolute %s=%q: %v", rootEnvName, root, err)
	}
	info, err := os.Lstat(absRoot)
	if err != nil {
		t.Fatalf("lstat %s=%q: %v", rootEnvName, root, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("%s=%q must not be a symlink", rootEnvName, root)
	}
	requirePrivateFixturePath(t, absRoot, info, rootEnvName)

	resolvedRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		t.Fatalf("resolve realpath %s=%q: %v", rootEnvName, root, err)
	}
	resolvedRoot = filepath.Clean(resolvedRoot)
	if resolvedRoot == "." || resolvedRoot == string(os.PathSeparator) {
		t.Fatalf("%s=%q resolves to unsafe root %q", rootEnvName, root, resolvedRoot)
	}
	if resolvedRoot != absRoot {
		t.Skipf("%s=%q must not include symlinked parent path components; resolved to %q", rootEnvName, root, resolvedRoot)
	}
	return resolvedRoot
}

// requireNoSymlinkPathComponents rejects symlinked path components before privileged fixture writes.
func requireNoSymlinkPathComponents(t *testing.T, root, relativePath, envName string) {
	t.Helper()

	current := root
	for _, part := range strings.Split(relativePath, string(os.PathSeparator)) {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			t.Fatalf("lstat %s candidate %q: %v", envName, current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			t.Fatalf("%s=%q resolves through symlinked component %q", envName, relativePath, current)
		}
		requirePrivateFixturePath(t, current, info, envName)

		resolvedCurrent, err := filepath.EvalSymlinks(current)
		if err != nil {
			t.Fatalf("resolve realpath for %s candidate %q: %v", envName, current, err)
		}
		resolvedCurrent = filepath.Clean(resolvedCurrent)
		componentRel, err := filepath.Rel(root, resolvedCurrent)
		if err != nil {
			t.Fatalf("verify %s candidate %q within %q: %v", envName, current, root, err)
		}
		if componentRel == ".." || strings.HasPrefix(componentRel, ".."+string(os.PathSeparator)) {
			t.Fatalf("%s=%q escapes verified root %q via %q", envName, relativePath, root, current)
		}
	}
}

// requireSandboxRunEvidence checks that the privileged run emitted mount/report evidence.
func requireSandboxRunEvidence(t *testing.T, output string) {
	t.Helper()

	for _, want := range []string{
		"the process is sandboxed",
		"source mount success!",
		"destination mount success!",
		"mount_info(",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("sync sandbox output missing %q evidence:\n%s", want, output)
		}
	}
}

// requireNoHostSyncerMountLeaks verifies the host mount table has no leftover syncer mounts.
func requireNoHostSyncerMountLeaks(t *testing.T) {
	t.Helper()

	findmnt, err := exec.LookPath("findmnt")
	if err != nil {
		t.Skipf("findmnt unavailable for host cleanup evidence: %v", err)
	}
	out, err := exec.Command(findmnt).CombinedOutput()
	if err != nil {
		t.Fatalf("collect host mount cleanup evidence: %v\n%s", err, out)
	}
	if strings.Contains(string(out), "syncer-") || strings.Contains(string(out), "fast-volume-syncer") {
		t.Fatalf("host mount table still contains syncer cleanup evidence:\n%s", out)
	}
}

// requirePrivateFixturePath ensures fixture directories are not writable by other local users.
func requirePrivateFixturePath(t *testing.T, path string, info os.FileInfo, envName string) {
	t.Helper()

	if info.Mode().Perm()&0o022 != 0 {
		t.Skipf("%s path %q must not be group/world writable for privileged fixture setup", envName, path)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		t.Skipf("could not inspect owner for %s path %q", envName, path)
	}
	if int(stat.Uid) != os.Geteuid() {
		t.Skipf("%s path %q must be owned by the privileged test user uid %d, got uid %d", envName, path, os.Geteuid(), stat.Uid)
	}
}

// sha256File returns a hex checksum for a copied fixture file.
func sha256File(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum[:]), nil
}
