package flow

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccComputeNetworkInterface_Basic(t *testing.T) {
	name := acctest.RandomWithPrefix("test-network-interface")
	server := testAccServerConfig(t, name, "10.102.0.0/24")

	testAccSequential(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: server + fmt.Sprintf(testAccComputeNetworkInterfaceConfigBasic, name, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_compute_network_interface.foobar", "id"),
					resource.TestCheckResourceAttrPair("flow_compute_network_interface.foobar", "server_id", "flow_compute_server.foobar", "id"),
					resource.TestCheckResourceAttrPair("flow_compute_network_interface.foobar", "network_id", "flow_compute_network.back", "id"),
					resource.TestCheckResourceAttrSet("flow_compute_network_interface.foobar", "private_ip"),
					resource.TestCheckResourceAttrSet("flow_compute_network_interface.foobar", "mac_address"),
					resource.TestCheckResourceAttrSet("flow_compute_network_interface.foobar", "security"),
				),
			},
			{
				Config: server + fmt.Sprintf(testAccComputeNetworkInterfaceConfigBasic, name, "security = false"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("flow_compute_network_interface.foobar", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("flow_compute_network_interface.foobar", "security", "false"),
					resource.TestCheckResourceAttr("flow_compute_network_interface.foobar", "security_group_ids.#", "0"),
				),
			},
			{
				ResourceName:      "flow_compute_network_interface.foobar",
				ImportState:       true,
				ImportStateIdFunc: testAccCompositeImportID("flow_compute_network_interface.foobar", "server_id", "id"),
				ImportStateVerify: true,
			},
			{
				Config: server + fmt.Sprintf(testAccComputeNetworkInterfaceConfigBasic, name, "security = false") + testAccComputeNetworkInterfaceConfigDataSource,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.flow_compute_network_interface.by_id", "id", "flow_compute_network_interface.foobar", "id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_network_interface.by_id", "server_id", "flow_compute_network_interface.foobar", "server_id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_network_interface.by_id", "network_id", "flow_compute_network_interface.foobar", "network_id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_network_interface.by_id", "private_ip", "flow_compute_network_interface.foobar", "private_ip"),
					resource.TestCheckResourceAttrPair("data.flow_compute_network_interface.by_id", "mac_address", "flow_compute_network_interface.foobar", "mac_address"),
					resource.TestCheckResourceAttrPair("data.flow_compute_network_interface.by_id", "security_group_ids.#", "flow_compute_network_interface.foobar", "security_group_ids.#"),
					resource.TestCheckResourceAttrPair("data.flow_compute_network_interface.by_id", "security", "flow_compute_network_interface.foobar", "security"),
					resource.TestCheckResourceAttrPair("data.flow_compute_network_interface.by_network", "id", "flow_compute_network_interface.foobar", "id"),
				),
			},
		},
	})
}

const testAccComputeNetworkInterfaceConfigBasic = `
resource "flow_compute_network" "back" {
	name        = "%s-back"
	location_id = 1
	cidr        = "10.102.1.0/24"
}

resource "flow_compute_network_interface" "foobar" {
	server_id  = flow_compute_server.foobar.id
	network_id = flow_compute_network.back.id

	%s
}
`

// testAccImportStateIDFunc builds a composite import id ("42:7") from the given attributes of a resource in the state

const testAccComputeNetworkInterfaceConfigDataSource = `
data "flow_compute_network_interface" "by_id" {
	server_id = flow_compute_network_interface.foobar.server_id
	id        = flow_compute_network_interface.foobar.id
}

data "flow_compute_network_interface" "by_network" {
	server_id  = flow_compute_network_interface.foobar.server_id
	network_id = flow_compute_network_interface.foobar.network_id
}
`
