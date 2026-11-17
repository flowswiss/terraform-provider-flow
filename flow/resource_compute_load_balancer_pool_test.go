package flow

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccComputeLoadBalancerPool_Basic(t *testing.T) {
	name := acctest.RandomWithPrefix("test-load-balancer-pool")

	testAccSequential(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccComputeLoadBalancerConfigBasic, name, "10.106.0.0/24") + testAccComputeLoadBalancerPoolConfig(true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_compute_load_balancer_pool.foobar", "id"),
					resource.TestCheckResourceAttrSet("flow_compute_load_balancer_pool.foobar", "name"),
					resource.TestCheckResourceAttrPair("flow_compute_load_balancer_pool.foobar", "load_balancer_id", "flow_compute_load_balancer.foobar", "id"),
					resource.TestCheckResourceAttrPair("flow_compute_load_balancer_pool.foobar", "entry_protocol_id", "data.flow_compute_load_balancer_protocol.http", "id"),
					resource.TestCheckResourceAttrPair("flow_compute_load_balancer_pool.foobar", "target_protocol_id", "data.flow_compute_load_balancer_protocol.http", "id"),
					resource.TestCheckResourceAttrPair("flow_compute_load_balancer_pool.foobar", "balancing_algorithm_id", "data.flow_compute_load_balancer_algorithm.round_robin", "id"),
					resource.TestCheckResourceAttr("flow_compute_load_balancer_pool.foobar", "entry_port", "80"),
					resource.TestCheckResourceAttr("flow_compute_load_balancer_pool.foobar", "sticky_session", "true"),
					resource.TestCheckResourceAttrPair("flow_compute_load_balancer_pool.foobar", "health_check.type_id", "data.flow_compute_load_balancer_health_check_type.http", "id"),
					resource.TestCheckResourceAttr("flow_compute_load_balancer_pool.foobar", "health_check.http.method", "GET"),
					resource.TestCheckResourceAttr("flow_compute_load_balancer_pool.foobar", "health_check.http.path", "/"),
					resource.TestCheckResourceAttr("flow_compute_load_balancer_pool.foobar", "health_check.interval", "10s"),
					resource.TestCheckResourceAttr("flow_compute_load_balancer_pool.foobar", "health_check.timeout", "5s"),
					resource.TestCheckResourceAttr("flow_compute_load_balancer_pool.foobar", "health_check.healthy_threshold", "2"),
					resource.TestCheckResourceAttr("flow_compute_load_balancer_pool.foobar", "health_check.unhealthy_threshold", "2"),
				),
			},
			{
				Config: fmt.Sprintf(testAccComputeLoadBalancerConfigBasic, name, "10.106.0.0/24") + testAccComputeLoadBalancerPoolConfig(false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("flow_compute_load_balancer_pool.foobar", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("flow_compute_load_balancer_pool.foobar", "sticky_session", "false"),
					resource.TestCheckResourceAttr("flow_compute_load_balancer_pool.foobar", "health_check.interval", "10s"),
					resource.TestCheckResourceAttr("flow_compute_load_balancer_pool.foobar", "health_check.timeout", "5s"),
				),
			},
			{
				ResourceName:      "flow_compute_load_balancer_pool.foobar",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccCompositeImportID("flow_compute_load_balancer_pool.foobar", "load_balancer_id", "id"),
			},
			{
				Config: fmt.Sprintf(testAccComputeLoadBalancerConfigBasic, name, "10.106.0.0/24") + testAccComputeLoadBalancerPoolConfig(false) + testAccComputeLoadBalancerPoolConfigDataSource,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.flow_compute_load_balancer_pool.by_id", "id", "flow_compute_load_balancer_pool.foobar", "id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_load_balancer_pool.by_id", "load_balancer_id", "flow_compute_load_balancer_pool.foobar", "load_balancer_id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_load_balancer_pool.by_id", "name", "flow_compute_load_balancer_pool.foobar", "name"),
					resource.TestCheckResourceAttrPair("data.flow_compute_load_balancer_pool.by_id", "balancing_algorithm_id", "flow_compute_load_balancer_pool.foobar", "balancing_algorithm_id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_load_balancer_pool.by_id", "sticky_session", "flow_compute_load_balancer_pool.foobar", "sticky_session"),
					resource.TestCheckResourceAttrPair("data.flow_compute_load_balancer_pool.by_id", "entry_protocol_id", "flow_compute_load_balancer_pool.foobar", "entry_protocol_id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_load_balancer_pool.by_id", "entry_port", "flow_compute_load_balancer_pool.foobar", "entry_port"),
					resource.TestCheckResourceAttrPair("data.flow_compute_load_balancer_pool.by_id", "target_protocol_id", "flow_compute_load_balancer_pool.foobar", "target_protocol_id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_load_balancer_pool.by_id", "certificate_id", "flow_compute_load_balancer_pool.foobar", "certificate_id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_load_balancer_pool.by_id", "health_check.type_id", "flow_compute_load_balancer_pool.foobar", "health_check.type_id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_load_balancer_pool.by_id", "health_check.http.method", "flow_compute_load_balancer_pool.foobar", "health_check.http.method"),
					resource.TestCheckResourceAttrPair("data.flow_compute_load_balancer_pool.by_id", "health_check.http.path", "flow_compute_load_balancer_pool.foobar", "health_check.http.path"),
					resource.TestCheckResourceAttrPair("data.flow_compute_load_balancer_pool.by_id", "health_check.interval", "flow_compute_load_balancer_pool.foobar", "health_check.interval"),
					resource.TestCheckResourceAttrPair("data.flow_compute_load_balancer_pool.by_id", "health_check.timeout", "flow_compute_load_balancer_pool.foobar", "health_check.timeout"),
					resource.TestCheckResourceAttrPair("data.flow_compute_load_balancer_pool.by_id", "health_check.healthy_threshold", "flow_compute_load_balancer_pool.foobar", "health_check.healthy_threshold"),
					resource.TestCheckResourceAttrPair("data.flow_compute_load_balancer_pool.by_id", "health_check.unhealthy_threshold", "flow_compute_load_balancer_pool.foobar", "health_check.unhealthy_threshold"),
					resource.TestCheckResourceAttrPair("data.flow_compute_load_balancer_pool.by_port", "id", "flow_compute_load_balancer_pool.foobar", "id"),
				),
			},
		},
	})
}

const testAccComputeLoadBalancerPoolConfigDataSource = `
data "flow_compute_load_balancer_pool" "by_id" {
	load_balancer_id = flow_compute_load_balancer_pool.foobar.load_balancer_id
	id               = flow_compute_load_balancer_pool.foobar.id
}

data "flow_compute_load_balancer_pool" "by_port" {
	load_balancer_id = flow_compute_load_balancer_pool.foobar.load_balancer_id
	entry_port       = flow_compute_load_balancer_pool.foobar.entry_port
}
`

func testAccComputeLoadBalancerPoolConfig(stickySession bool) string {
	return fmt.Sprintf(`
data "flow_compute_load_balancer_algorithm" "round_robin" {
	key = "round_robin"
}

data "flow_compute_load_balancer_protocol" "http" {
	key = "http"
}

data "flow_compute_load_balancer_health_check_type" "http" {
	key = "http"
}

resource "flow_compute_load_balancer_pool" "foobar" {
	load_balancer_id       = flow_compute_load_balancer.foobar.id
	entry_protocol_id      = data.flow_compute_load_balancer_protocol.http.id
	entry_port             = 80
	target_protocol_id     = data.flow_compute_load_balancer_protocol.http.id
	balancing_algorithm_id = data.flow_compute_load_balancer_algorithm.round_robin.id
	sticky_session         = %t

	health_check = {
		type_id             = data.flow_compute_load_balancer_health_check_type.http.id
		http                = { method = "GET", path = "/" }
		interval            = "10s"
		timeout             = "5s"
		healthy_threshold   = 2
		unhealthy_threshold = 2
	}
}
`, stickySession)
}
