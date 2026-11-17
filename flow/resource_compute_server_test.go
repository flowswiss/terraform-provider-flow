package flow

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccComputeServer_Basic(t *testing.T) {
	serverName := acctest.RandomWithPrefix("test-server")
	config := testAccServerConfig(t, serverName, "10.101.0.0/24")

	// key pair and network carry the same name and replace on a rename, so only the server's line changes
	serverLine := `name        = "` + serverName + `"
	location_id = 1
	image_id`
	if strings.Count(config, serverLine) != 1 {
		t.Fatal("server name line not found in testAccServerConfig")
	}
	renamed := strings.Replace(config, serverLine, strings.Replace(serverLine, serverName, serverName+"-renamed", 1), 1)

	testAccSequential(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_compute_server.foobar", "id"),
					resource.TestCheckResourceAttr("flow_compute_server.foobar", "name", serverName),
					resource.TestCheckResourceAttr("flow_compute_server.foobar", "location_id", "1"),
					resource.TestCheckResourceAttrPair("flow_compute_server.foobar", "image_id", "data.flow_compute_image.ubuntu", "id"),
					resource.TestCheckResourceAttrPair("flow_compute_server.foobar", "product_id", "data.flow_product.small", "id"),
					resource.TestCheckResourceAttrPair("flow_compute_server.foobar", "network_id", "flow_compute_network.foobar", "id"),
					resource.TestCheckResourceAttrPair("flow_compute_server.foobar", "key_pair_id", "flow_compute_key_pair.foobar", "id"),
					resource.TestCheckResourceAttrSet("flow_compute_server.foobar", "private_ip"),
					resource.TestCheckResourceAttrSet("flow_compute_server.foobar", "network_interface_id"),
				),
			},
			{
				Config: renamed,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("flow_compute_server.foobar", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("flow_compute_server.foobar", "name", serverName+"-renamed"),
				),
			},
			{
				ResourceName:      "flow_compute_server.foobar",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: renamed + testAccComputeServerConfigDataSource,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.flow_compute_server.by_id", "id", "flow_compute_server.foobar", "id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_server.by_id", "name", "flow_compute_server.foobar", "name"),
					resource.TestCheckResourceAttrPair("data.flow_compute_server.by_id", "location_id", "flow_compute_server.foobar", "location_id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_server.by_id", "image_id", "flow_compute_server.foobar", "image_id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_server.by_id", "product_id", "flow_compute_server.foobar", "product_id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_server.by_id", "key_pair_id", "flow_compute_server.foobar", "key_pair_id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_server.by_name", "id", "flow_compute_server.foobar", "id"),
				),
			},
		},
	})
}

const testAccComputeServerConfigDataSource = `
data "flow_compute_server" "by_id" {
	id = flow_compute_server.foobar.id
}

data "flow_compute_server" "by_name" {
	name = flow_compute_server.foobar.name
}
`
