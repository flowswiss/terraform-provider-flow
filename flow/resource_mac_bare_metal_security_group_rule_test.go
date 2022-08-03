package flow

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccMacBareMetalSecurityGroupRule_Basic(t *testing.T) {
	securityGroupName := acctest.RandomWithPrefix("test-security-group")

	protocolNumber := "6"
	protocolName := "tcp"
	fromPort := 22
	toPort := 22
	ipRange := "1.1.1.1/32"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccMacBareMetalSecurityGroupRuleConfigBasic, securityGroupName, protocolName, fromPort, toPort, ipRange),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_mac_bare_metal_security_group_rule.foobar", "id"),
					resource.TestCheckResourceAttr("flow_mac_bare_metal_security_group_rule.foobar", "direction", "ingress"),
					resource.TestCheckResourceAttr("flow_mac_bare_metal_security_group_rule.foobar", "protocol.number", protocolNumber),
					resource.TestCheckResourceAttr("flow_mac_bare_metal_security_group_rule.foobar", "protocol.name", protocolName),
					resource.TestCheckResourceAttr("flow_mac_bare_metal_security_group_rule.foobar", "port_range.from", fmt.Sprint(fromPort)),
					resource.TestCheckResourceAttr("flow_mac_bare_metal_security_group_rule.foobar", "port_range.to", fmt.Sprint(toPort)),
					resource.TestCheckResourceAttr("flow_mac_bare_metal_security_group_rule.foobar", "ip_range", ipRange),
					resource.TestCheckNoResourceAttr("flow_mac_bare_metal_security_group_rule.foobar", "icmp"),
				),
			},
		},
	})
}

const testAccMacBareMetalSecurityGroupRuleConfigBasic = `
data "flow_location" "zrh1" {
	name = "ZRH1"
}

data "flow_mac_bare_metal_network" "foobar" {
	location_id = data.flow_location.zrh1.id
}

resource "flow_mac_bare_metal_security_group" "foobar" {
	name        = "%s"
	network_id = data.flow_mac_bare_metal_network.foobar.id
}

resource "flow_mac_bare_metal_security_group_rule" "foobar" {
	security_group_id = flow_mac_bare_metal_security_group.foobar.id

	direction = "ingress"
	protocol  = { name = "%s" }

	port_range = {
		from = %d
		to   = %d
	}

	ip_range = "%s"
}
`
