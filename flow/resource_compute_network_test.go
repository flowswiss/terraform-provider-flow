package flow

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccComputeNetwork_Basic(t *testing.T) {
	networkName := acctest.RandomWithPrefix("test-network")
	networkCIDR := "192.168.1.0/24"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccComputeNetworkConfigBasic, networkName, networkCIDR),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_compute_network.foobar", "id"),
					resource.TestCheckResourceAttr("flow_compute_network.foobar", "name", networkName),
					resource.TestCheckResourceAttr("flow_compute_network.foobar", "cidr", networkCIDR),
					resource.TestCheckResourceAttr("flow_compute_network.foobar", "location_id", "1"),
					resource.TestCheckResourceAttr("flow_compute_network.foobar", "domain_name_servers.0", "1.1.1.1"),
					resource.TestCheckResourceAttr("flow_compute_network.foobar", "domain_name_servers.1", "8.8.8.8"),
					resource.TestCheckResourceAttrSet("flow_compute_network.foobar", "allocation_pool.start"),
					resource.TestCheckResourceAttrSet("flow_compute_network.foobar", "allocation_pool.end"),
					resource.TestCheckResourceAttrSet("flow_compute_network.foobar", "gateway_ip"),
				),
			},
		},
	})
}

const testAccComputeNetworkConfigBasic = `
resource "flow_compute_network" "foobar" {
	name        = "%s"
	cidr 	    = "%s"
	location_id = 1
}
`
