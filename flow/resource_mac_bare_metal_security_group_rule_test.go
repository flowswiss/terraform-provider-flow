package flow

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccMacBareMetalSecurityGroupRule_Basic(t *testing.T) {
	networkName := acctest.RandomWithPrefix("test-network")
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
				Config: fmt.Sprintf(testAccMacBareMetalSecurityGroupRuleConfigBasic, networkName, securityGroupName, "foobar_ingress", "ingress", protocolName, fromPort, toPort, ipRange),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_mac_bare_metal_security_group_rule.foobar_ingress", "id"),
					resource.TestCheckResourceAttr("flow_mac_bare_metal_security_group_rule.foobar_ingress", "direction", "ingress"),
					resource.TestCheckResourceAttr("flow_mac_bare_metal_security_group_rule.foobar_ingress", "protocol.number", protocolNumber),
					resource.TestCheckResourceAttr("flow_mac_bare_metal_security_group_rule.foobar_ingress", "protocol.name", protocolName),
					resource.TestCheckResourceAttr("flow_mac_bare_metal_security_group_rule.foobar_ingress", "port_range.from", fmt.Sprint(fromPort)),
					resource.TestCheckResourceAttr("flow_mac_bare_metal_security_group_rule.foobar_ingress", "port_range.to", fmt.Sprint(toPort)),
					resource.TestCheckResourceAttr("flow_mac_bare_metal_security_group_rule.foobar_ingress", "ip_range", ipRange),
					resource.TestCheckNoResourceAttr("flow_mac_bare_metal_security_group_rule.foobar_ingress", "icmp"),
				),
			},
			{
				Config: fmt.Sprintf(testAccMacBareMetalSecurityGroupRuleConfigBasic, networkName, securityGroupName, "foobar_egress", "egress", protocolName, fromPort, toPort, ipRange),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_mac_bare_metal_security_group_rule.foobar_egress", "id"),
					resource.TestCheckResourceAttr("flow_mac_bare_metal_security_group_rule.foobar_egress", "direction", "egress"),
					resource.TestCheckResourceAttr("flow_mac_bare_metal_security_group_rule.foobar_egress", "protocol.number", protocolNumber),
					resource.TestCheckResourceAttr("flow_mac_bare_metal_security_group_rule.foobar_egress", "protocol.name", protocolName),
					resource.TestCheckResourceAttr("flow_mac_bare_metal_security_group_rule.foobar_egress", "port_range.from", fmt.Sprint(fromPort)),
					resource.TestCheckResourceAttr("flow_mac_bare_metal_security_group_rule.foobar_egress", "port_range.to", fmt.Sprint(toPort)),
					resource.TestCheckResourceAttr("flow_mac_bare_metal_security_group_rule.foobar_egress", "ip_range", ipRange),
					resource.TestCheckNoResourceAttr("flow_mac_bare_metal_security_group_rule.foobar_egress", "icmp"),
				),
			},
		},
	})
}

const testAccMacBareMetalSecurityGroupRuleConfigBasic = `
data "flow_location" "zrh1" {
	name = "ZRH1"
}

resource "flow_mac_bare_metal_network" "foobar" {
	name        = "%s"
	location_id = data.flow_location.zrh1.id
}

resource "flow_mac_bare_metal_security_group" "foobar" {
	name        = "%s"
	network_id = flow_mac_bare_metal_network.foobar.id
}

resource "flow_mac_bare_metal_security_group_rule" "%s" {
	security_group_id = flow_mac_bare_metal_security_group.foobar.id

	direction = "%s"
	protocol  = { name = "%s" }

	port_range = {
		from = %d
		to   = %d
	}

	ip_range = "%s"
}
`
