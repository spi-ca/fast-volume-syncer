package selector

import (
	"bytes"
	"context"
	"log"
	"testing"
)

func TestRunner_loadCopyEntryCSV(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testData := []byte(
		`node,src_vol,dst_vol,src_path,dst_path,project_id,project_name,used_size,used_size_human,volume_type,volume_size,volume_size_human
0,00000000-0000-4000-8000-000000000000,vol_fixture-00000000-0000-4000-8000-000000000000,/projects/project-a,project/PLM,106,org-kep-nlp,19526478175232,17.76Ti,project,35184372088832,32Ti
1,storage-a,vol_fixture-00000000-0000-4000-8000-000000000000,/sandbox/fixture-data/units/unit-a/sandbox/session-a,sandbox/user_at_example_invalid,58,org-adrec,16397493867520,14.914Ti,sandbox,35184372088832,32Ti
2,storage-a,vol_fixture-00000000-0000-4000-8000-000000000000,/sandbox/fixture-data/units/unit-a/sandbox/session-a,sandbox/user_at_example_invalid,72,org-recoteam-experiment,15961047820288,14.517Ti,sandbox,35184372088832,32Ti
3,storage-a,vol_fixture-00000000-0000-4000-8000-000000000000,/fixture/ro/fixture-data/units/58/project/project-a,project/adrec_gpu,58,org-adrec,3025672800256,2.752Ti,project,35184372088832,32Ti
`)
	r := Runner{}

	entries := r.loadCopyEntryCSV(ctx, bytes.NewReader(testData))
	for entry := range entries {
		log.Printf("%v", entry)
	}
}
