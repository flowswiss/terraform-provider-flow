package flow

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccComputeLoadBalancerMember_Basic(t *testing.T) {
	name := acctest.RandomWithPrefix("test-load-balancer-member")

	testAccSequential(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServerConfig(t, name, "10.107.0.0/24") + testAccComputeLoadBalancerMemberConfigBasic,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_compute_load_balancer_member.foobar", "id"),
					resource.TestCheckResourceAttr("flow_compute_load_balancer_member.foobar", "name", "server-1"),
					resource.TestCheckResourceAttrPair("flow_compute_load_balancer_member.foobar", "load_balancer_id", "flow_compute_load_balancer.foobar", "id"),
					resource.TestCheckResourceAttrPair("flow_compute_load_balancer_member.foobar", "pool_id", "flow_compute_load_balancer_pool.foobar", "id"),
					resource.TestCheckResourceAttrPair("flow_compute_load_balancer_member.foobar", "address", "flow_compute_server.foobar", "private_ip"),
					resource.TestCheckResourceAttr("flow_compute_load_balancer_member.foobar", "port", "8080"),
				),
			},
			{
				ResourceName:      "flow_compute_load_balancer_member.foobar",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccCompositeImportID("flow_compute_load_balancer_member.foobar", "load_balancer_id", "pool_id", "id"),
				// the health check moves status on its own, nothing listens on the member's port
				ImportStateVerifyIgnore: []string{"status"},
			},
			{
				Config: testAccServerConfig(t, name, "10.107.0.0/24") + testAccComputeLoadBalancerMemberConfigBasic + testAccComputeLoadBalancerMemberConfigDataSource,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.flow_compute_load_balancer_member.by_id", "id", "flow_compute_load_balancer_member.foobar", "id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_load_balancer_member.by_id", "pool_id", "flow_compute_load_balancer_member.foobar", "pool_id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_load_balancer_member.by_id", "load_balancer_id", "flow_compute_load_balancer_member.foobar", "load_balancer_id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_load_balancer_member.by_id", "name", "flow_compute_load_balancer_member.foobar", "name"),
					resource.TestCheckResourceAttrPair("data.flow_compute_load_balancer_member.by_id", "address", "flow_compute_load_balancer_member.foobar", "address"),
					resource.TestCheckResourceAttrPair("data.flow_compute_load_balancer_member.by_id", "port", "flow_compute_load_balancer_member.foobar", "port"),
					resource.TestCheckResourceAttrSet("data.flow_compute_load_balancer_member.by_id", "status"),
					resource.TestCheckResourceAttrPair("data.flow_compute_load_balancer_member.by_name", "id", "flow_compute_load_balancer_member.foobar", "id"),
				),
			},
		},
	})
}

const testAccComputeLoadBalancerMemberConfigDataSource = `
data "flow_compute_load_balancer_member" "by_id" {
	load_balancer_id = flow_compute_load_balancer_member.foobar.load_balancer_id
	pool_id          = flow_compute_load_balancer_member.foobar.pool_id
	id               = flow_compute_load_balancer_member.foobar.id
}

data "flow_compute_load_balancer_member" "by_name" {
	load_balancer_id = flow_compute_load_balancer_member.foobar.load_balancer_id
	pool_id          = flow_compute_load_balancer_member.foobar.pool_id
	name             = flow_compute_load_balancer_member.foobar.name
}
`

var testAccComputeLoadBalancerMemberConfigBasic = `
resource "flow_compute_load_balancer" "foobar" {
	name        = flow_compute_server.foobar.name
	location_id = 1
	network_id  = flow_compute_network.foobar.id
}
` + testAccComputeLoadBalancerPoolConfig(true) + `
resource "flow_compute_load_balancer_member" "foobar" {
	load_balancer_id = flow_compute_load_balancer.foobar.id
	pool_id          = flow_compute_load_balancer_pool.foobar.id
	name             = "server-1"
	address          = flow_compute_server.foobar.private_ip
	port             = 8080
}
`
