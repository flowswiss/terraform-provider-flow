package flow

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccComputeRouterInterface_Basic(t *testing.T) {
	networkName := acctest.RandomWithPrefix("test-network")
	networkCIDR := "192.168.1.0/24"
	routerName := acctest.RandomWithPrefix("test-router")

	testAccSequential(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccComputeRouterInterfaceConfigBasic, networkName, networkCIDR, routerName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_compute_router_interface.foobar", "id"),
					resource.TestCheckResourceAttrSet("flow_compute_router_interface.foobar", "router_id"),
					resource.TestCheckResourceAttrSet("flow_compute_router_interface.foobar", "network_id"),
					resource.TestCheckResourceAttrSet("flow_compute_router_interface.foobar", "private_ip"),
				),
			},
			{
				ResourceName:      "flow_compute_router_interface.foobar",
				ImportState:       true,
				ImportStateIdFunc: testAccCompositeImportID("flow_compute_router_interface.foobar", "router_id", "id"),
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(testAccComputeRouterInterfaceConfigBasic, networkName, networkCIDR, routerName) + testAccComputeRouterInterfaceConfigDataSource,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.flow_compute_router_interface.by_id", "id", "flow_compute_router_interface.foobar", "id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_router_interface.by_id", "router_id", "flow_compute_router_interface.foobar", "router_id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_router_interface.by_id", "network_id", "flow_compute_router_interface.foobar", "network_id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_router_interface.by_id", "private_ip", "flow_compute_router_interface.foobar", "private_ip"),
					resource.TestCheckResourceAttrPair("data.flow_compute_router_interface.by_network", "id", "flow_compute_router_interface.foobar", "id"),
				),
			},
		},
	})
}

// testAccCompositeImportID builds the colon separated import id of a resource
// that is only addressable through its parent, e.g. `router_id:id`

const testAccComputeRouterInterfaceConfigBasic = `
locals {
	location_id = 1
}

resource "flow_compute_network" "foobar" {
	name        = "%s"
	location_id = local.location_id

	cidr = "%s"
}

resource "flow_compute_router" "foobar" {
	name        = "%s"
	location_id = local.location_id

	public = false
}

resource "flow_compute_router_interface" "foobar" {
	router_id = flow_compute_router.foobar.id
	network_id = flow_compute_network.foobar.id
}
`

const testAccComputeRouterInterfaceConfigDataSource = `
data "flow_compute_router_interface" "by_id" {
	router_id = flow_compute_router_interface.foobar.router_id
	id        = flow_compute_router_interface.foobar.id
}

data "flow_compute_router_interface" "by_network" {
	router_id  = flow_compute_router_interface.foobar.router_id
	network_id = flow_compute_router_interface.foobar.network_id
}
`
