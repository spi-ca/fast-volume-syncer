package selector

import (
	"bytes"
	"context"
	"testing"
)

const copyEntryCSVFixture = `node,src_vol,dst_vol,src_path,dst_path,project_id,project_name,used_size,used_size_human,volume_type,volume_size,volume_size_human,destination_project_name,volume_name,source_volume_key
7, storage-a , vol-a , /src/a , /dst/a ,1, project-a ,1024,1Ki,project,2048,2Ki, dest-a , volume-a , key-a
5,storage-b,vol-b,/src/b,/dst/b,2,project-b,4096,4Ki,sandbox,8192,8Ki,dest-b,volume-b,key-b
bad,storage-c,vol-c,/src/c,/dst/c,3,project-c,1,1B,project,2,2B,dest-c,volume-c,key-c
3,too,few,columns
8,storage-d,vol-d,/src/d,/dst/d,not-int,project-d,1,1B,project,2,2B,dest-d,volume-d,key-d
9,storage-e,vol-e,/src/e,/dst/e,4,project-e,not-int,1B,project,2,2B,dest-e,volume-e,key-e
10,storage-f,vol-f,/src/f,/dst/f,5,project-f,1,1B,project,not-int,2B,dest-f,volume-f,key-f
`

func collectCopyEntries(t *testing.T, r Runner, data string) []copyEntry {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	entryChan := make(chan copyEntry)
	go r.loadCopyEntryCSV(ctx, bytes.NewBufferString(data), entryChan)

	var entries []copyEntry
	for entry := range entryChan {
		entries = append(entries, entry)
	}
	return entries
}

func TestRunnerLoadCopyEntryCSVSkipsHeaderMalformedRowsAndTrimsFields(t *testing.T) {
	entries := collectCopyEntries(t, Runner{NodeSelector: -1}, copyEntryCSVFixture)
	if len(entries) != 2 {
		t.Fatalf("expected 2 valid entries, got %d: %#v", len(entries), entries)
	}

	first := entries[0]
	if first.Node != 7 {
		t.Fatalf("expected first node 7, got %d", first.Node)
	}
	if first.SourceVolume != "storage-a" || first.DestinationVolume != "vol-a" {
		t.Fatalf("expected trimmed volumes, got source=%q destination=%q", first.SourceVolume, first.DestinationVolume)
	}
	if first.SourcePath != "/src/a" || first.DestinationPath != "/dst/a" {
		t.Fatalf("expected trimmed paths, got source=%q destination=%q", first.SourcePath, first.DestinationPath)
	}
	if first.SourceProjectId != 1 || first.UsedSize != 1024 || first.VolumeSize != 2048 {
		t.Fatalf("unexpected parsed numeric fields: %#v", first)
	}

	second := entries[1]
	if second.Node != 5 || second.VolumeType != "sandbox" || second.SourceVolumeKey != "key-b" {
		t.Fatalf("unexpected second entry: %#v", second)
	}
}

func TestRunnerLoadCopyEntryCSVAppliesNodeSelector(t *testing.T) {
	entries := collectCopyEntries(t, Runner{NodeSelector: 5}, copyEntryCSVFixture)
	if len(entries) != 1 {
		t.Fatalf("expected one selected entry, got %d: %#v", len(entries), entries)
	}
	if entries[0].Node != 5 || entries[0].SourceProjectName != "project-b" {
		t.Fatalf("unexpected selected entry: %#v", entries[0])
	}
}

func TestRunnerLoadCopyEntryCSVReturnsNoEntriesWhenSelectorDoesNotMatch(t *testing.T) {
	entries := collectCopyEntries(t, Runner{NodeSelector: 42}, copyEntryCSVFixture)
	if len(entries) != 0 {
		t.Fatalf("expected no entries, got %d: %#v", len(entries), entries)
	}
}
