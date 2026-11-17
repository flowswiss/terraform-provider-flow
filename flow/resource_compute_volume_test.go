package flow

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccComputeVolume_Basic(t *testing.T) {
	volumeName := acctest.RandomWithPrefix("test-volume")
	volumeSize := acctest.RandIntRange(1, 20)

	testAccSequential(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccComputeVolumeConfigBasic, volumeName, volumeSize),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_compute_volume.foobar", "id"),
					resource.TestCheckResourceAttr("flow_compute_volume.foobar", "name", volumeName),
					resource.TestCheckResourceAttr("flow_compute_volume.foobar", "location_id", "1"),
					resource.TestCheckResourceAttr("flow_compute_volume.foobar", "size", fmt.Sprint(volumeSize)),
					resource.TestCheckResourceAttrSet("flow_compute_volume.foobar", "serial_number"),
					resource.TestCheckNoResourceAttr("flow_compute_volume.foobar", "restore_from_snapshot_id"),
				),
			},
			{
				Config: fmt.Sprintf(testAccComputeVolumeConfigBasic, volumeName+"-renamed", volumeSize+1),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("flow_compute_volume.foobar", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("flow_compute_volume.foobar", "name", volumeName+"-renamed"),
					resource.TestCheckResourceAttr("flow_compute_volume.foobar", "size", fmt.Sprint(volumeSize+1)),
				),
			},
			{
				ResourceName:      "flow_compute_volume.foobar",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(testAccComputeVolumeConfigBasic, volumeName+"-renamed", volumeSize+1) + testAccComputeVolumeConfigDataSource,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.flow_compute_volume.by_id", "id", "flow_compute_volume.foobar", "id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_volume.by_id", "serial_number", "flow_compute_volume.foobar", "serial_number"),
					resource.TestCheckResourceAttrPair("data.flow_compute_volume.by_id", "name", "flow_compute_volume.foobar", "name"),
					resource.TestCheckResourceAttrPair("data.flow_compute_volume.by_id", "size", "flow_compute_volume.foobar", "size"),
					resource.TestCheckResourceAttrPair("data.flow_compute_volume.by_id", "location_id", "flow_compute_volume.foobar", "location_id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_volume.by_name", "id", "flow_compute_volume.foobar", "id"),
				),
			},
		},
	})
}

const testAccComputeVolumeConfigBasic = `
resource "flow_compute_volume" "foobar" {
	name        = "%s"
	location_id = 1

	size = %d
}
`

const testAccComputeVolumeConfigDataSource = `
data "flow_compute_volume" "by_id" {
	id = flow_compute_volume.foobar.id
}

data "flow_compute_volume" "by_name" {
	name = flow_compute_volume.foobar.name
}
`
