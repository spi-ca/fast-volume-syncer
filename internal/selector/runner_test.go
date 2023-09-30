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
		`node,src_vol,dst_vol,src_path,dst_path,project_id,project_name,used_size,used_size_human,volume_type,volume_size,volume_size_human,destination_project_name,volume_name,source_volume_key
7,storage-a,vol_fixture-00000000-0000-4000-8000-000000000000,/fixture/ro/fixture-data/units/1/project/project-a,/,1,exampleorg,7060507722752,6.422Ti,project,10000000000000,10Ti,exampleorg/Project-Alpha,worker-60,exampleorg/project/project-a
0,storage-a,vol_fixture-00000000-0000-4000-8000-000000000000,/fixture/ro/fixture-data/units/1/project/project-a,/,1,exampleorg,4759618989056,4.329Ti,project,7000000000000,7Ti,exampleorg/Project-Beta,worker-48,exampleorg/project/project-a
3,00000000-0000-4000-8000-000000000000,vol_fixture-00000000-0000-4000-8000-000000000000,/projects/project-a,/,228,example-metrics,24727327058944,22.49Ti,project,33000000000000,31Ti,example-metrics,worker-74,example-metrics/projects/project-a
5,00000000-0000-4000-8000-000000000000,vol_fixture-00000000-0000-4000-8000-000000000000,/sandboxes/session-a,/,155,example-chat,39155129344,36.467Gi,sandbox,100000000000,94Gi,example-chat,worker-13,example-chat/sandboxes/session-a
5,00000000-0000-4000-8000-000000000000,vol_fixture-00000000-0000-4000-8000-000000000000,/sandboxes/session-a,/,159,example-video,55811019776,51.979Gi,sandbox,100000000000,94Gi,example-video,worker-15,example-video/sandboxes/session-a
5,00000000-0000-4000-8000-000000000000,vol_fixture-00000000-0000-4000-8000-000000000000,/sandboxes/session-a,sandbox/user_at_example_invalid,85,exampleorg-dataset-manager,111616,109.0Ki,sandbox,100000000000,94Gi,exampleorg,worker-39,exampleorg-dataset-manager/sandboxes/session-a
5,00000000-0000-4000-8000-000000000000,vol_fixture-00000000-0000-4000-8000-000000000000,/sandboxes/session-a,sandbox/user_at_example_invalid,85,exampleorg-dataset-manager,1431264256,1.333Gi,sandbox,100000000000,94Gi,exampleorg,worker-39,exampleorg-dataset-manager/sandboxes/session-a
5,00000000-0000-4000-8000-000000000000,vol_fixture-00000000-0000-4000-8000-000000000000,/sandboxes/session-a,sandbox/user_at_example_invalid,232,example-optimize,35729408,34.075Mi,sandbox,100000000000,94Gi,example-optimize,worker-16,example-optimize/sandboxes/session-a
`)
	r := Runner{}
	entryChan := make(chan copyEntry)
	go r.loadCopyEntryCSV(ctx, bytes.NewReader(testData), entryChan)
	for entry := range entryChan {
		log.Printf("%v", entry)
	}
}
