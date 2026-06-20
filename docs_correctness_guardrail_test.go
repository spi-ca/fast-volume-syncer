// Package main contains repository-level documentation correctness guardrails.
package main

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// readTextFile loads a repository document and fails the guardrail test on read errors.
func readTextFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

// requireDocText asserts that a document still contains required operational guidance.
func requireDocText(t *testing.T, path string, wants ...string) {
	t.Helper()
	text := readTextFile(t, path)
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Fatalf("%s missing required correctness guardrail text %q", path, want)
		}
	}
}

// requireCommand runs a lightweight validation command that protects docs-only baseline checks.
func requireCommand(t *testing.T, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %s failed: %v\n%s", name, strings.Join(args, " "), err, out)
	}
}

// requireNFSVerificationClaimsLinked prevents unsupported NFS verification claims from appearing without evidence links.
func requireNFSVerificationClaimsLinked(t *testing.T) {
	t.Helper()

	claimPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bnfs\b.*\b(verified|verification\s+(?:complete|completed)|validated|confirmed)\b`),
		regexp.MustCompile(`(?i)\b(verified|validated|confirmed)\b.*\bnfs\b`),
		regexp.MustCompile(`(?i)nfs.*검증\s*(완료|됨|했다|되었|확인)`),
		regexp.MustCompile(`(?i)검증\s*(완료|됨|했다|되었|확인).*nfs`),
	}
	allowedLinkPattern := regexp.MustCompile(`\[[^\]]+\]\([^)]*(docs/)?(nfs-sync-sandbox-evidence\.md|evidence/nfs-sync-sandbox\.example\.md)[^)]*\)`)
	paths := []string{"README.md", "AGENTS.md", "CLAUDE.md"}
	if err := filepath.WalkDir("docs", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path == "docs/guidelines" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ".md" {
			paths = append(paths, path)
		}
		return nil
	}); err != nil {
		t.Fatalf("walk docs for NFS verification claims: %v", err)
	}

	for _, path := range paths {
		for lineNumber, line := range strings.Split(readTextFile(t, path), "\n") {
			claimsVerified := false
			for _, pattern := range claimPatterns {
				if pattern.MatchString(line) {
					claimsVerified = true
					break
				}
			}
			if !claimsVerified {
				continue
			}
			if !containsEvidenceMarkdownLink(line, allowedLinkPattern) {
				t.Fatalf("%s:%d NFS verification claim must use a Markdown link to the runbook or evidence template: %q", path, lineNumber+1, line)
			}
		}
	}
}

// containsEvidenceMarkdownLink reports whether a line has a non-image Markdown evidence link.
func containsEvidenceMarkdownLink(line string, pattern *regexp.Regexp) bool {
	for _, match := range pattern.FindAllStringIndex(line, -1) {
		if match[0] == 0 || line[match[0]-1] != '!' {
			return true
		}
	}
	return false
}

// TestDocumentationCorrectnessGuardrails keeps docs aligned with validation and integration-test limits.
func TestDocumentationCorrectnessGuardrails(t *testing.T) {
	requireNFSVerificationClaimsLinked(t)
	requireDocText(t, "docs/requirements.md",
		"Privileged operations such as mounting are runtime/environment requirements, not unit-test assumptions.",
	)
	requireDocText(t, "docs/test-and-benchmark-gaps.md",
		"Do not claim benchmark results without raw command output.",
		"Do not treat privileged mount or network storage behavior as verified by unit tests alone.",
		"go test -tags=integration -run TestBwrapCopyE2E -count=1 .",
		"go test -tags='integration,nfs' -run TestNFSSyncSandboxE2E -count=1 .",
		"correctness-evidence.md",
	)
	requireDocText(t, "README.md",
		"직접 실행한 `sync`는 기본적으로 namespace 샌드박스가 아니며",
		"selector/daemon이 띄운 Linux child",
		"scripts/check-go-comments.py",
		"go test -tags=integration -run '^$' .",
		"go test -tags='integration,nfs' -run '^$' .",
		"Mermaid SVG/2x PNG",
	)
	requireDocText(t, "AGENTS.md",
		"scripts/check-go-comments.py",
		"go test -tags=integration -run '^$' .",
		"go test -tags='integration,nfs' -run '^$' .",
		"Mermaid SVG/2x PNG",
	)
	requireDocText(t, "CLAUDE.md",
		"scripts/check-go-comments.py",
		"go test -tags=integration -run '^$' .",
		"go test -tags='integration,nfs' -run '^$' .",
		"Mermaid SVG/2x PNG",
	)
	requireDocText(t, "docs/operations.md",
		"go test -tags=integration -run TestBwrapCopyE2E -count=1 .",
		"go test -tags='integration,nfs' -run TestNFSSyncSandboxE2E -count=1 .",
		"scripts/check-go-comments.py",
		"npx --yes @mermaid-js/mermaid-cli",
		"bwrap",
		"correctness-evidence.md",
		"Direct `sync` runs are not namespace-sandboxed by default",
	)
	requireDocText(t, "docs/diagrams/README.md",
		"npx --yes @mermaid-js/mermaid-cli",
		"PNG artifacts are exported at 2x scale",
	)
	requireDocText(t, "docs/diagrams/validation-checks.mmd",
		"scripts/check-go-comments.py",
		"go test -tags=integration -run '^$' .",
		"go test -tags='integration,nfs' -run '^$' .",
		"Mermaid render",
		"code --> fmt --> comments --> tagged --> tests",
		"integration --> fmt --> comments --> tagged --> tests",
		"integration --> vet",
		"diagrams --> render --> tests",
	)
	requireDocText(t, "docs/correctness-evidence.md",
		"Privileged mount and network-storage behavior is environment-dependent.",
		"sandbox_bwrap_integration_test.go` must stay behind `//go:build integration && linux`",
		"nfs_sync_sandbox_linux_integration_test.go` must stay behind `//go:build integration && nfs && linux`",
		"Benchmark claims require raw local output.",
		"nfs-sync-sandbox-evidence.md",
		"use the selector-launched sync child path",
		"FVS_TEST_SRC_NFS_HOST",
		"FVS_TEST_DST_VERIFY_ROOT",
		"dummy non-secret `source_volume_key`",
		"redact `source_volume_key`",
		"go test ./...",
		"go test -tags=integration -run TestBwrapCopyE2E -count=1 .",
		"go test -tags='integration,nfs' -run TestNFSSyncSandboxE2E -count=1 .",
	)
	requireDocText(t, "docs/nfs-sync-sandbox-evidence.md",
		"Unit tests and bwrap copy smoke tests are not sufficient evidence",
		"Run the real selector-launched child path",
		"Do not simulate it by setting `_SYNCER_INVOKED` or `_SYNCER_SANDBOXED` on a direct `sync` command",
		"FVS_TEST_SRC_NFS_HOST",
		"FVS_TEST_SRC_NFS_EXPORT",
		"FVS_TEST_DST_NFS_HOST",
		"FVS_TEST_DST_NFS_EXPORT",
		"FVS_TEST_SRC_SUBPATH",
		"FVS_TEST_DST_SUBPATH",
		"FVS_TEST_SRC_VERIFY_ROOT",
		"FVS_TEST_DST_VERIFY_ROOT",
		"Source verify root (`FVS_TEST_SRC_VERIFY_ROOT`):",
		"Destination verify root (`FVS_TEST_DST_VERIFY_ROOT`):",
		"Verify roots: private, disposable, owned by privileged test user, not group/world writable, no symlinked path components",
		"dummy non-secret `source_volume_key`",
		"redact `source_volume_key`",
		"select 0",
		"one-row.csv",
		"sudo env -i",
		"Use a private `0700` workspace for the binary, CSV, and log",
		"Do not use `sudo -E` or `go run` for privileged evidence collection",
		"findmnt",
		"If any required evidence is missing, report NFS/mount sandbox behavior as unverified.",
	)
	requireDocText(t, "docs/evidence/nfs-sync-sandbox.example.md",
		"Source verify root (`FVS_TEST_SRC_VERIFY_ROOT`):",
		"Destination verify root (`FVS_TEST_DST_VERIFY_ROOT`):",
		"Verify roots: private, disposable, owned by privileged test user, not group/world writable, no symlinked path components",
		"sudo env -i",
		"/path/to/fast-volume-syncer select 0 /path/to/one-row.csv",
		"Redactions applied (`source_volume_key`, credentials, tokens, unrelated secrets):",
	)

	integrationTest := readTextFile(t, "sandbox_bwrap_integration_test.go")
	if !strings.HasPrefix(integrationTest, "//go:build integration && linux\n") {
		t.Fatalf("sandbox_bwrap_integration_test.go must stay behind the integration && linux build tag")
	}
	if !strings.Contains(integrationTest, "bwrap") {
		t.Fatalf("sandbox_bwrap_integration_test.go must exercise bwrap")
	}

	nfsIntegrationTest := readTextFile(t, "nfs_sync_sandbox_linux_integration_test.go")
	if !strings.HasPrefix(nfsIntegrationTest, "//go:build integration && nfs && linux\n") {
		t.Fatalf("nfs_sync_sandbox_linux_integration_test.go must stay behind the integration && nfs && linux build tag")
	}
	for _, want := range []string{
		"FVS_TEST_SRC_NFS_HOST",
		"FVS_TEST_SRC_NFS_EXPORT",
		"FVS_TEST_DST_NFS_HOST",
		"FVS_TEST_DST_NFS_EXPORT",
		"FVS_TEST_SRC_SUBPATH",
		"FVS_TEST_DST_SUBPATH",
		"FVS_TEST_SRC_VERIFY_ROOT",
		"FVS_TEST_DST_VERIFY_ROOT",
		"TestNFSSyncSandboxE2E",
		"clear source fixture root",
		"prepare source fixture root",
		"the process is sandboxed",
		"writeNFSSelectorCSV",
		"one-row.csv",
		"\"select\"",
	} {
		if !strings.Contains(nfsIntegrationTest, want) {
			t.Fatalf("nfs_sync_sandbox_linux_integration_test.go missing required integration guardrail text %q", want)
		}
	}
	for _, reject := range []string{
		"\"_SYNCER_INVOKED=true\"",
		"\"_SYNCER_SANDBOXED=true\"",
	} {
		if strings.Contains(nfsIntegrationTest, reject) {
			t.Fatalf("nfs_sync_sandbox_linux_integration_test.go must not inject %s directly; use selector-launched child env", reject)
		}
	}

	requireCommand(t, "scripts/check-go-comments.py")
	requireCommand(t, "go", "test", "-tags=integration", "-run", "^$", ".")
	requireCommand(t, "go", "test", "-tags=integration,nfs", "-run", "^$", ".")
}
