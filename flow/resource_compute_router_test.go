package flow

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccComputeRouter_Basic(t *testing.T) {
	routerName := acctest.RandomWithPrefix("test-router")

	testAccSequential(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccComputeRouterConfigBasic, "foobar_public", routerName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_compute_router.foobar_public", "id"),
					resource.TestCheckResourceAttr("flow_compute_router.foobar_public", "name", routerName),
					resource.TestCheckResourceAttr("flow_compute_router.foobar_public", "location_id", "1"),
					resource.TestCheckResourceAttr("flow_compute_router.foobar_public", "public", "true"),
					resource.TestCheckResourceAttrSet("flow_compute_router.foobar_public", "public_ip"),
				),
			},
			{
				Config: fmt.Sprintf(testAccComputeRouterConfigBasic, "foobar_private", routerName, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_compute_router.foobar_private", "id"),
					resource.TestCheckResourceAttr("flow_compute_router.foobar_private", "name", routerName),
					resource.TestCheckResourceAttr("flow_compute_router.foobar_private", "location_id", "1"),
					resource.TestCheckResourceAttr("flow_compute_router.foobar_private", "public", "false"),
					resource.TestCheckNoResourceAttr("flow_compute_router.foobar_private", "public_ip"),
				),
			},
		},
	})
}

func TestAccComputeRouter_PublicOff(t *testing.T) {
	routerName := acctest.RandomWithPrefix("test-router")

	testAccSequential(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccComputeRouterConfigBasic, "foobar", routerName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("flow_compute_router.foobar", "public", "true"),
					resource.TestCheckResourceAttrSet("flow_compute_router.foobar", "public_ip"),
				),
			},
			{
				Config: fmt.Sprintf(testAccComputeRouterConfigBasic, "foobar", routerName, false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("flow_compute_router.foobar", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("flow_compute_router.foobar", "public", "false"),
					resource.TestCheckNoResourceAttr("flow_compute_router.foobar", "public_ip"),
				),
			},
			{
				ResourceName:      "flow_compute_router.foobar",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(testAccComputeRouterConfigBasic, "foobar", routerName, false) + testAccComputeRouterConfigDataSource,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.flow_compute_router.by_id", "id", "flow_compute_router.foobar", "id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_router.by_id", "name", "flow_compute_router.foobar", "name"),
					resource.TestCheckResourceAttrPair("data.flow_compute_router.by_id", "location_id", "flow_compute_router.foobar", "location_id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_router.by_id", "public", "flow_compute_router.foobar", "public"),
					resource.TestCheckResourceAttrPair("data.flow_compute_router.by_id", "public_ip", "flow_compute_router.foobar", "public_ip"),
					resource.TestCheckResourceAttrPair("data.flow_compute_router.by_name", "id", "flow_compute_router.foobar", "id"),
				),
			},
		},
	})
}

const testAccComputeRouterConfigBasic = `
resource "flow_compute_router" "%s" {
	name        = "%s"
	location_id = 1

	public = %t
}
`

const testAccComputeRouterConfigDataSource = `
data "flow_compute_router" "by_id" {
	id = flow_compute_router.foobar.id
}

data "flow_compute_router" "by_name" {
	name = flow_compute_router.foobar.name
}
`
