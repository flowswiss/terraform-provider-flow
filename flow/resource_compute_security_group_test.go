package flow

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccComputeSecurityGroup_Basic(t *testing.T) {
	securityGroupName := acctest.RandomWithPrefix("test-security-group")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccComputeSecurityGroupConfigBasic, securityGroupName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_compute_security_group.foobar", "id"),
					resource.TestCheckResourceAttr("flow_compute_security_group.foobar", "name", securityGroupName),
					resource.TestCheckResourceAttr("flow_compute_security_group.foobar", "location_id", "1"),
				),
			},
		},
	})
}

const testAccComputeSecurityGroupConfigBasic = `
resource "flow_compute_security_group" "foobar" {
	name        = "%s"
	location_id = 1
}
`
