package flow

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccMacBareMetalElasticIP_Basic(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMacBareMetalElasticIPConfigBasic,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_mac_bare_metal_elastic_ip.foobar", "id"),
					resource.TestCheckResourceAttrSet("flow_mac_bare_metal_elastic_ip.foobar", "public_ip"),
					resource.TestCheckResourceAttr("flow_mac_bare_metal_elastic_ip.foobar", "location_id", "2"),
				),
			},
		},
	})
}

const testAccMacBareMetalElasticIPConfigBasic = `
data "flow_location" "zrh1" {
	name = "ZRH1"
}

resource "flow_mac_bare_metal_elastic_ip" "foobar" {
	location_id = data.flow_location.zrh1.id
}
`
