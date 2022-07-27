package flow

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccMacBareMetalSecurityGroup_Basic(t *testing.T) {
	networkName := acctest.RandomWithPrefix("test-network")
	securityGroupName := acctest.RandomWithPrefix("test-security-group")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccMacBareMetalSecurityGroupConfigBasic, networkName, securityGroupName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_mac_bare_metal_security_group.foobar", "id"),
					resource.TestCheckResourceAttr("flow_mac_bare_metal_security_group.foobar", "name", securityGroupName),
					resource.TestCheckResourceAttrSet("flow_mac_bare_metal_security_group.foobar", "network_id"),
				),
			},
		},
	})
}

const testAccMacBareMetalSecurityGroupConfigBasic = `
data "flow_location" "zrh1" {
	name = "ZRH1"
}

resource "flow_mac_bare_metal_network" "foobar" {
	name        = "%s"
	location_id = data.flow_location.zrh1.id
}

resource "flow_mac_bare_metal_security_group" "foobar" {
	name        = "%s"
	network_id = flow_mac_bare_metal_network.foobar.id
}
`
