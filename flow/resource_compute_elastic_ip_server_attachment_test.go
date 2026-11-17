package flow

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccComputeElasticIPServerAttachment_Basic(t *testing.T) {
	name := acctest.RandomWithPrefix("test-elastic-ip-attachment")
	server := testAccServerConfig(t, name, "10.104.0.0/24")

	testAccSequential(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: server + fmt.Sprintf(testAccComputeElasticIPServerAttachmentConfigBasic, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("flow_compute_elastic_ip_server_attachment.foobar", "server_id", "flow_compute_server.foobar", "id"),
					resource.TestCheckResourceAttrPair("flow_compute_elastic_ip_server_attachment.foobar", "network_interface_id", "flow_compute_server.foobar", "network_interface_id"),
					resource.TestCheckResourceAttrPair("flow_compute_elastic_ip_server_attachment.foobar", "elastic_ip_id", "flow_compute_elastic_ip.foobar", "id"),
				),
			},
			{
				ResourceName:                         "flow_compute_elastic_ip_server_attachment.foobar",
				ImportState:                          true,
				ImportStateIdFunc:                    testAccCompositeImportID("flow_compute_elastic_ip_server_attachment.foobar", "server_id", "elastic_ip_id"),
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "server_id",
			},
		},
	})
}

const testAccComputeElasticIPServerAttachmentConfigBasic = `
resource "flow_compute_router" "foobar" {
	name        = "%s"
	location_id = 1
	public      = true
}

resource "flow_compute_router_interface" "foobar" {
	router_id  = flow_compute_router.foobar.id
	network_id = flow_compute_network.foobar.id
}

resource "flow_compute_elastic_ip" "foobar" {
	location_id = 1
}

resource "flow_compute_elastic_ip_server_attachment" "foobar" {
	server_id            = flow_compute_server.foobar.id
	network_interface_id = flow_compute_server.foobar.network_interface_id
	elastic_ip_id        = flow_compute_elastic_ip.foobar.id

	depends_on = [flow_compute_router_interface.foobar]
}
`
