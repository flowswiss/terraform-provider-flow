package flow

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccComputeSnapshot_Basic(t *testing.T) {

	volumeName := acctest.RandomWithPrefix("test-volume")
	volumeSize := acctest.RandIntRange(1, 20)
	snapshotName := acctest.RandomWithPrefix("test-snapshot")

	testAccSequential(t, resource.TestCase{
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
			{
				Config: fmt.Sprintf(testAccComputeSnapshotConfigBasic, volumeName, volumeSize, snapshotName+"-renamed"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("flow_compute_snapshot.foobar", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("flow_compute_snapshot.foobar", "name", snapshotName+"-renamed"),
				),
			},
			{
				ResourceName:      "flow_compute_snapshot.foobar",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(testAccComputeSnapshotConfigBasic, volumeName, volumeSize, snapshotName+"-renamed") + testAccComputeSnapshotConfigDataSource,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.flow_compute_snapshot.by_id", "id", "flow_compute_snapshot.foobar", "id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_snapshot.by_id", "name", "flow_compute_snapshot.foobar", "name"),
					resource.TestCheckResourceAttrPair("data.flow_compute_snapshot.by_id", "volume_id", "flow_compute_snapshot.foobar", "volume_id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_snapshot.by_id", "size", "flow_compute_snapshot.foobar", "size"),
					resource.TestCheckResourceAttrPair("data.flow_compute_snapshot.by_id", "created_at", "flow_compute_snapshot.foobar", "created_at"),
					resource.TestCheckResourceAttrPair("data.flow_compute_snapshot.by_name", "id", "flow_compute_snapshot.foobar", "id"),
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

const testAccComputeSnapshotConfigDataSource = `
data "flow_compute_snapshot" "by_id" {
	id = flow_compute_snapshot.foobar.id
}

data "flow_compute_snapshot" "by_name" {
	name = flow_compute_snapshot.foobar.name
}
`
