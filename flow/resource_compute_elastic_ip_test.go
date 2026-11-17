package flow

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccComputeElasticIP_Basic(t *testing.T) {
	testAccSequential(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccComputeElasticIPConfigBasic,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_compute_elastic_ip.foobar", "id"),
					resource.TestCheckResourceAttrSet("flow_compute_elastic_ip.foobar", "public_ip"),
					resource.TestCheckResourceAttr("flow_compute_elastic_ip.foobar", "location_id", "1"),
				),
			},
			{
				ResourceName:      "flow_compute_elastic_ip.foobar",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccComputeElasticIPConfigBasic + testAccComputeElasticIPConfigDataSource,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.flow_compute_elastic_ip.by_id", "id", "flow_compute_elastic_ip.foobar", "id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_elastic_ip.by_id", "location_id", "flow_compute_elastic_ip.foobar", "location_id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_elastic_ip.by_id", "public_ip", "flow_compute_elastic_ip.foobar", "public_ip"),
					resource.TestCheckResourceAttrPair("data.flow_compute_elastic_ip.by_public_ip", "id", "flow_compute_elastic_ip.foobar", "id"),
				),
			},
		},
	})
}

const testAccComputeElasticIPConfigBasic = `
resource "flow_compute_elastic_ip" "foobar" {
	location_id = 1
}
`

const testAccComputeElasticIPConfigDataSource = `
data "flow_compute_elastic_ip" "by_id" {
	id = flow_compute_elastic_ip.foobar.id
}

data "flow_compute_elastic_ip" "by_public_ip" {
	public_ip = flow_compute_elastic_ip.foobar.public_ip
}
`
