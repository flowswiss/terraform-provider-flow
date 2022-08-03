package flow

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccMacBareMetalNetwork_Basic(t *testing.T) {
	t.Skip("api does currently not allow creating multiple mac bare metal networks. enable this once it is allowed")

	networkName := acctest.RandomWithPrefix("test-network")
	domainName := "example.com"
	domainNameServer := "1.1.2.2"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccMacBareMetalNetworkConfigBasic, networkName, domainName, domainNameServer),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_mac_bare_metal_network.foobar", "id"),
					resource.TestCheckResourceAttr("flow_mac_bare_metal_network.foobar", "name", networkName),
					resource.TestCheckResourceAttrSet("flow_mac_bare_metal_network.foobar", "cidr"),
					resource.TestCheckResourceAttr("flow_mac_bare_metal_network.foobar", "location_id", "2"),
					resource.TestCheckResourceAttr("flow_mac_bare_metal_network.foobar", "domain_name", domainName),
					resource.TestCheckResourceAttr("flow_mac_bare_metal_network.foobar", "domain_name_servers.0", domainNameServer),
					resource.TestCheckResourceAttrSet("flow_mac_bare_metal_network.foobar", "allocation_pool.start"),
					resource.TestCheckResourceAttrSet("flow_mac_bare_metal_network.foobar", "allocation_pool.end"),
					resource.TestCheckResourceAttrSet("flow_mac_bare_metal_network.foobar", "gateway_ip"),
				),
			},
		},
	})
}

const testAccMacBareMetalNetworkConfigBasic = `
resource "flow_mac_bare_metal_network" "foobar" {
	name        = "%s"
	location_id = 2

	domain_name = "%s"
	domain_name_servers = [
		"%s"
	]
}
`
