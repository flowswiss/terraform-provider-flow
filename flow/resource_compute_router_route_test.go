package flow

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccComputeRouterRoute_Basic(t *testing.T) {
	networkName := acctest.RandomWithPrefix("test-network")
	networkCIDR := "192.168.1.0/24"
	routerName := acctest.RandomWithPrefix("test-router")

	destination := "0.0.0.0/0"
	nextHop, err := acctest.RandIpAddress(networkCIDR)
	if err != nil {
		t.Fatal(err)
	}

	testAccSequential(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccComputeRouterRouteConfigBasic, networkName, networkCIDR, routerName, destination, nextHop),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_compute_router_route.foobar", "id"),
					resource.TestCheckResourceAttrSet("flow_compute_router_route.foobar", "router_id"),
					resource.TestCheckResourceAttr("flow_compute_router_route.foobar", "destination", destination),
					resource.TestCheckResourceAttr("flow_compute_router_route.foobar", "next_hop", nextHop),
				),
			},
			{
				ResourceName:      "flow_compute_router_route.foobar",
				ImportState:       true,
				ImportStateIdFunc: testAccCompositeImportID("flow_compute_router_route.foobar", "router_id", "id"),
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(testAccComputeRouterRouteConfigBasic, networkName, networkCIDR, routerName, destination, nextHop) + testAccComputeRouterRouteConfigDataSource,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.flow_compute_router_route.by_id", "id", "flow_compute_router_route.foobar", "id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_router_route.by_id", "router_id", "flow_compute_router_route.foobar", "router_id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_router_route.by_id", "destination", "flow_compute_router_route.foobar", "destination"),
					resource.TestCheckResourceAttrPair("data.flow_compute_router_route.by_id", "next_hop", "flow_compute_router_route.foobar", "next_hop"),
					resource.TestCheckResourceAttrPair("data.flow_compute_router_route.by_destination", "id", "flow_compute_router_route.foobar", "id"),
				),
			},
		},
	})
}

const testAccComputeRouterRouteConfigBasic = `
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

resource "flow_compute_router_route" "foobar" {
	router_id = flow_compute_router.foobar.id
	destination = "%s"
	next_hop = "%s"

	depends_on = [flow_compute_router_interface.foobar]
}
`

const testAccComputeRouterRouteConfigDataSource = `
data "flow_compute_router_route" "by_id" {
	router_id = flow_compute_router_route.foobar.router_id
	id        = flow_compute_router_route.foobar.id
}

data "flow_compute_router_route" "by_destination" {
	router_id   = flow_compute_router_route.foobar.router_id
	destination = flow_compute_router_route.foobar.destination
}
`
