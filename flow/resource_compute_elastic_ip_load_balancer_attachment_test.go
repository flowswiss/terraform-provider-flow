package flow

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccComputeElasticIPLoadBalancerAttachment_Basic(t *testing.T) {
	name := acctest.RandomWithPrefix("test-lb-elastic-ip-attachment")

	testAccSequential(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccComputeElasticIPLoadBalancerAttachmentConfigBasic, name, "10.106.0.0/24"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("flow_compute_elastic_ip_load_balancer_attachment.foobar", "load_balancer_id", "flow_compute_load_balancer.foobar", "id"),
					resource.TestCheckResourceAttrPair("flow_compute_elastic_ip_load_balancer_attachment.foobar", "elastic_ip_id", "flow_compute_elastic_ip.foobar", "id"),
				),
			},
			{
				Config: fmt.Sprintf(testAccComputeElasticIPLoadBalancerAttachmentConfigBasic, name, "10.106.0.0/24"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("flow_compute_load_balancer.foobar", "public_ip", "flow_compute_elastic_ip.foobar", "public_ip"),
				),
			},
			{
				ResourceName:                         "flow_compute_elastic_ip_load_balancer_attachment.foobar",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "load_balancer_id",
				ImportStateIdFunc:                    testAccCompositeImportID("flow_compute_elastic_ip_load_balancer_attachment.foobar", "load_balancer_id", "elastic_ip_id"),
			},
		},
	})
}

const testAccComputeElasticIPLoadBalancerAttachmentConfigBasic = `
resource "flow_compute_network" "foobar" {
	name        = "%[1]s"
	location_id = 1
	cidr        = "%[2]s"
}

resource "flow_compute_router" "foobar" {
	name        = "%[1]s"
	location_id = 1
	public      = true
}

resource "flow_compute_router_interface" "foobar" {
	router_id  = flow_compute_router.foobar.id
	network_id = flow_compute_network.foobar.id
}

resource "flow_compute_load_balancer" "foobar" {
	name        = "%[1]s"
	location_id = 1
	network_id  = flow_compute_network.foobar.id
}

resource "flow_compute_elastic_ip" "foobar" {
	location_id = 1
}

resource "flow_compute_elastic_ip_load_balancer_attachment" "foobar" {
	load_balancer_id = flow_compute_load_balancer.foobar.id
	elastic_ip_id    = flow_compute_elastic_ip.foobar.id

	depends_on = [flow_compute_router_interface.foobar]
}
`
