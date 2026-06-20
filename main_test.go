package main

import (
	"os/exec"
	"strings"
	"testing"
)

func runCLIExpectFailure(t *testing.T, args ...string) string {
	t.Helper()

	cmdArgs := append([]string{"run", "."}, args...)
	cmd := exec.Command("go", cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected go run . %v to fail, output:\n%s", args, out)
	}
	return string(out)
}

func TestUsageDocumentsSelectorPlaceholderForms(t *testing.T) {
	output := runCLIExpectFailure(t)
	for _, want := range []string{
		"fast-volume-syncer copy SRC_PATH DST_PATH",
		"fast-volume-syncer sync SRC_PATH [SRC_SUBPATH] DST_PATH [DST_SUBPATH]",
		"fast-volume-syncer select [NODE_SELECTOR:-1]",
		"fast-volume-syncer select _|NODE_SELECTOR COPY_INFO_CSV_PATH:data/09_copy_entries.csv",
		"fast-volume-syncer start [NODE_SELECTOR:-1|_]",
		"fast-volume-syncer start _|NODE_SELECTOR COPY_INFO_CSV_PATH:data/09_copy_entries.csv",
		"fast-volume-syncer stop",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("usage output missing %q:\n%s", want, output)
		}
	}
}

func TestSelectRejectsBareCSVPathAndDocumentsUsage(t *testing.T) {
	output := runCLIExpectFailure(t, "select", "custom.csv")
	for _, want := range []string{
		"failed to parse nodeSelector",
		"fast-volume-syncer select _|NODE_SELECTOR COPY_INFO_CSV_PATH:data/09_copy_entries.csv",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("select custom.csv output missing %q:\n%s", want, output)
		}
	}
}

func TestParseSelectorArgs(t *testing.T) {
	tests := []struct {
		name                string
		args                []string
		allowBareUnderscore bool
		wantSelector        int
		wantCSV             string
		wantErr             bool
	}{
		{name: "default", wantSelector: defaultNodeSelector, wantCSV: defaultCSVFilename},
		{name: "selector only", args: []string{"7"}, wantSelector: 7, wantCSV: defaultCSVFilename},
		{name: "selector and csv", args: []string{"7", "custom.csv"}, wantSelector: 7, wantCSV: "custom.csv"},
		{name: "placeholder and csv", args: []string{"_", "custom.csv"}, wantSelector: defaultNodeSelector, wantCSV: "custom.csv"},
		{name: "select rejects bare placeholder", args: []string{"_"}, wantErr: true},
		{name: "start accepts bare placeholder", args: []string{"_"}, allowBareUnderscore: true, wantSelector: defaultNodeSelector, wantCSV: defaultCSVFilename},
		{name: "bare csv rejected", args: []string{"custom.csv"}, wantErr: true},
		{name: "too many args rejected", args: []string{"1", "custom.csv", "extra"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSelector, gotCSV, err := parseSelectorArgs(tt.args, tt.allowBareUnderscore)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got selector=%d csv=%q", gotSelector, gotCSV)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotSelector != tt.wantSelector || gotCSV != tt.wantCSV {
				t.Fatalf("expected selector=%d csv=%q, got selector=%d csv=%q", tt.wantSelector, tt.wantCSV, gotSelector, gotCSV)
			}
		})
	}
}
