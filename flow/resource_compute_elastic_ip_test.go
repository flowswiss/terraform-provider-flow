package flow

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccComputeElasticIP_Basic(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
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
		},
	})
}

const testAccComputeElasticIPConfigBasic = `
resource "flow_compute_elastic_ip" "foobar" {
	location_id = 1
}
`
