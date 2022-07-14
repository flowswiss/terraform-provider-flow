package flow

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccComputeRouterInterface_Basic(t *testing.T) {
	networkName := acctest.RandomWithPrefix("test-network")
	networkCIDR := "192.168.1.0/24"
	routerName := acctest.RandomWithPrefix("test-router")

	resource.ParallelTest(t, resource.TestCase{
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
		},
	})
}

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
