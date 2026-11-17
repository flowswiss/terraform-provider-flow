package flow

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccComputeNetwork_Basic(t *testing.T) {
	networkName := acctest.RandomWithPrefix("test-network")
	networkCIDR := "192.168.1.0/24"

	testAccSequential(t, resource.TestCase{
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
			{
				Config: fmt.Sprintf(testAccComputeNetworkConfigBasic, networkName+"-renamed", networkCIDR),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("flow_compute_network.foobar", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("flow_compute_network.foobar", "name", networkName+"-renamed"),
					resource.TestCheckResourceAttr("flow_compute_network.foobar", "cidr", networkCIDR),
				),
			},
			{
				ResourceName:      "flow_compute_network.foobar",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(testAccComputeNetworkConfigBasic, networkName+"-renamed", networkCIDR) + testAccComputeNetworkConfigDataSource,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.flow_compute_network.by_id", "id", "flow_compute_network.foobar", "id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_network.by_id", "name", "flow_compute_network.foobar", "name"),
					resource.TestCheckResourceAttrPair("data.flow_compute_network.by_id", "cidr", "flow_compute_network.foobar", "cidr"),
					resource.TestCheckResourceAttrPair("data.flow_compute_network.by_id", "location_id", "flow_compute_network.foobar", "location_id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_network.by_id", "domain_name_servers.#", "flow_compute_network.foobar", "domain_name_servers.#"),
					resource.TestCheckResourceAttrPair("data.flow_compute_network.by_id", "allocation_pool.start", "flow_compute_network.foobar", "allocation_pool.start"),
					resource.TestCheckResourceAttrPair("data.flow_compute_network.by_id", "allocation_pool.end", "flow_compute_network.foobar", "allocation_pool.end"),
					resource.TestCheckResourceAttrPair("data.flow_compute_network.by_id", "gateway_ip", "flow_compute_network.foobar", "gateway_ip"),
					resource.TestCheckResourceAttrPair("data.flow_compute_network.by_name", "id", "flow_compute_network.foobar", "id"),
				),
			},
		},
	})
}

const testAccComputeNetworkConfigDataSource = `
data "flow_compute_network" "by_id" {
	id = flow_compute_network.foobar.id
}

data "flow_compute_network" "by_name" {
	name = flow_compute_network.foobar.name
}
`

const testAccComputeNetworkConfigBasic = `
resource "flow_compute_network" "foobar" {
	name        = "%s"
	cidr 	    = "%s"
	location_id = 1
}
`
