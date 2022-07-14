package flow

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccComputeSnapshot_Basic(t *testing.T) {
	t.Skip("skipping test due to race condition during deletion in api")

	volumeName := acctest.RandomWithPrefix("test-volume")
	volumeSize := acctest.RandIntRange(1, 20)
	snapshotName := acctest.RandomWithPrefix("test-snapshot")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccComputeSnapshotConfigBasic, volumeName, volumeSize, snapshotName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_compute_snapshot.foobar", "id"),
					resource.TestCheckResourceAttr("flow_compute_snapshot.foobar", "name", snapshotName),
					resource.TestCheckResourceAttr("flow_compute_snapshot.foobar", "size", fmt.Sprint(volumeSize)),
					resource.TestCheckResourceAttrSet("flow_compute_snapshot.foobar", "volume_id"),
					resource.TestCheckResourceAttrSet("flow_compute_snapshot.foobar", "created_at"),
				),
			},
		},
	})
}

const testAccComputeSnapshotConfigBasic = `
resource "flow_compute_volume" "foobar" {
	name        = "%s"
	location_id = 1

	size = %d
}

resource "flow_compute_snapshot" "foobar" {
	name        = "%s"
	volume_id   = flow_compute_volume.foobar.id
}
`
