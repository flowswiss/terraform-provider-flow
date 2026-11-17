package flow

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccComputeSecurityGroupRule_Basic(t *testing.T) {
	securityGroupName := acctest.RandomWithPrefix("test-security-group")

	protocolNumber := "6"
	protocolName := "tcp"
	fromPort := 22
	toPort := 22
	ipRange := "1.1.1.1/32"

	testAccSequential(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccComputeSecurityGroupRuleConfigBasic, securityGroupName, "foobar_ingress", "ingress", protocolName, fromPort, toPort, ipRange),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_compute_security_group_rule.foobar_ingress", "id"),
					resource.TestCheckResourceAttr("flow_compute_security_group_rule.foobar_ingress", "direction", "ingress"),
					resource.TestCheckResourceAttr("flow_compute_security_group_rule.foobar_ingress", "protocol.number", protocolNumber),
					resource.TestCheckResourceAttr("flow_compute_security_group_rule.foobar_ingress", "protocol.name", protocolName),
					resource.TestCheckResourceAttr("flow_compute_security_group_rule.foobar_ingress", "port_range.from", fmt.Sprint(fromPort)),
					resource.TestCheckResourceAttr("flow_compute_security_group_rule.foobar_ingress", "port_range.to", fmt.Sprint(toPort)),
					resource.TestCheckResourceAttr("flow_compute_security_group_rule.foobar_ingress", "ip_range", ipRange),
					resource.TestCheckNoResourceAttr("flow_compute_security_group_rule.foobar_ingress", "icmp"),
					resource.TestCheckNoResourceAttr("flow_compute_security_group_rule.foobar_ingress", "remote_security_group_id"),
				),
			},
			{
				Config: fmt.Sprintf(testAccComputeSecurityGroupRuleConfigBasic, securityGroupName, "foobar_egress", "egress", protocolName, fromPort, toPort, ipRange),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_compute_security_group_rule.foobar_egress", "id"),
					resource.TestCheckResourceAttr("flow_compute_security_group_rule.foobar_egress", "direction", "egress"),
					resource.TestCheckResourceAttr("flow_compute_security_group_rule.foobar_egress", "protocol.number", protocolNumber),
					resource.TestCheckResourceAttr("flow_compute_security_group_rule.foobar_egress", "protocol.name", protocolName),
					resource.TestCheckResourceAttr("flow_compute_security_group_rule.foobar_egress", "port_range.from", fmt.Sprint(fromPort)),
					resource.TestCheckResourceAttr("flow_compute_security_group_rule.foobar_egress", "port_range.to", fmt.Sprint(toPort)),
					resource.TestCheckResourceAttr("flow_compute_security_group_rule.foobar_egress", "ip_range", ipRange),
					resource.TestCheckNoResourceAttr("flow_compute_security_group_rule.foobar_egress", "icmp"),
					resource.TestCheckNoResourceAttr("flow_compute_security_group_rule.foobar_egress", "remote_security_group_id"),
				),
			},
			{
				ResourceName:      "flow_compute_security_group_rule.foobar_egress",
				ImportState:       true,
				ImportStateIdFunc: testAccCompositeImportID("flow_compute_security_group_rule.foobar_egress", "security_group_id", "id"),
				ImportStateVerify: true,
			},
			{
				Config: fmt.Sprintf(testAccComputeSecurityGroupRuleConfigBasic, securityGroupName, "foobar_egress", "egress", protocolName, fromPort, toPort, ipRange) + fmt.Sprintf(testAccComputeSecurityGroupRuleConfigDataSource, "foobar_egress"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.flow_compute_security_group_rule.by_id", "id", "flow_compute_security_group_rule.foobar_egress", "id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_security_group_rule.by_id", "security_group_id", "flow_compute_security_group_rule.foobar_egress", "security_group_id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_security_group_rule.by_id", "direction", "flow_compute_security_group_rule.foobar_egress", "direction"),
					resource.TestCheckResourceAttrPair("data.flow_compute_security_group_rule.by_id", "protocol.number", "flow_compute_security_group_rule.foobar_egress", "protocol.number"),
					resource.TestCheckResourceAttrPair("data.flow_compute_security_group_rule.by_id", "protocol.name", "flow_compute_security_group_rule.foobar_egress", "protocol.name"),
					resource.TestCheckResourceAttrPair("data.flow_compute_security_group_rule.by_id", "port_range.from", "flow_compute_security_group_rule.foobar_egress", "port_range.from"),
					resource.TestCheckResourceAttrPair("data.flow_compute_security_group_rule.by_id", "port_range.to", "flow_compute_security_group_rule.foobar_egress", "port_range.to"),
					resource.TestCheckResourceAttrPair("data.flow_compute_security_group_rule.by_id", "icmp.type", "flow_compute_security_group_rule.foobar_egress", "icmp.type"),
					resource.TestCheckResourceAttrPair("data.flow_compute_security_group_rule.by_id", "icmp.code", "flow_compute_security_group_rule.foobar_egress", "icmp.code"),
					resource.TestCheckResourceAttrPair("data.flow_compute_security_group_rule.by_id", "ip_range", "flow_compute_security_group_rule.foobar_egress", "ip_range"),
					resource.TestCheckResourceAttrPair("data.flow_compute_security_group_rule.by_id", "remote_security_group_id", "flow_compute_security_group_rule.foobar_egress", "remote_security_group_id"),
				),
			},
		},
	})
}

func TestAccComputeSecurityGroupRule_ICMP(t *testing.T) {
	securityGroupName := acctest.RandomWithPrefix("test-security-group")

	testAccSequential(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccComputeSecurityGroupRuleConfigICMP, securityGroupName, 8, 0),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("flow_compute_security_group_rule.foobar_icmp", "id"),
					resource.TestCheckResourceAttr("flow_compute_security_group_rule.foobar_icmp", "protocol.number", "1"),
					resource.TestCheckResourceAttr("flow_compute_security_group_rule.foobar_icmp", "protocol.name", "icmp"),
					resource.TestCheckResourceAttr("flow_compute_security_group_rule.foobar_icmp", "icmp.type", "8"),
					resource.TestCheckResourceAttr("flow_compute_security_group_rule.foobar_icmp", "icmp.code", "0"),
					resource.TestCheckNoResourceAttr("flow_compute_security_group_rule.foobar_icmp", "port_range"),
				),
			},
			{
				Config: fmt.Sprintf(testAccComputeSecurityGroupRuleConfigICMP, securityGroupName, 0, 0),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("flow_compute_security_group_rule.foobar_icmp", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("flow_compute_security_group_rule.foobar_icmp", "icmp.type", "0"),
					resource.TestCheckResourceAttr("flow_compute_security_group_rule.foobar_icmp", "icmp.code", "0"),
				),
			},
			{
				Config: fmt.Sprintf(testAccComputeSecurityGroupRuleConfigICMP, securityGroupName, 0, 0) + fmt.Sprintf(testAccComputeSecurityGroupRuleConfigDataSource, "foobar_icmp"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.flow_compute_security_group_rule.by_id", "id", "flow_compute_security_group_rule.foobar_icmp", "id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_security_group_rule.by_id", "security_group_id", "flow_compute_security_group_rule.foobar_icmp", "security_group_id"),
					resource.TestCheckResourceAttrPair("data.flow_compute_security_group_rule.by_id", "direction", "flow_compute_security_group_rule.foobar_icmp", "direction"),
					resource.TestCheckResourceAttrPair("data.flow_compute_security_group_rule.by_id", "protocol.number", "flow_compute_security_group_rule.foobar_icmp", "protocol.number"),
					resource.TestCheckResourceAttrPair("data.flow_compute_security_group_rule.by_id", "protocol.name", "flow_compute_security_group_rule.foobar_icmp", "protocol.name"),
					resource.TestCheckResourceAttrPair("data.flow_compute_security_group_rule.by_id", "port_range.from", "flow_compute_security_group_rule.foobar_icmp", "port_range.from"),
					resource.TestCheckResourceAttrPair("data.flow_compute_security_group_rule.by_id", "port_range.to", "flow_compute_security_group_rule.foobar_icmp", "port_range.to"),
					resource.TestCheckResourceAttrPair("data.flow_compute_security_group_rule.by_id", "icmp.type", "flow_compute_security_group_rule.foobar_icmp", "icmp.type"),
					resource.TestCheckResourceAttrPair("data.flow_compute_security_group_rule.by_id", "icmp.code", "flow_compute_security_group_rule.foobar_icmp", "icmp.code"),
					resource.TestCheckResourceAttrPair("data.flow_compute_security_group_rule.by_id", "ip_range", "flow_compute_security_group_rule.foobar_icmp", "ip_range"),
					resource.TestCheckResourceAttrPair("data.flow_compute_security_group_rule.by_id", "remote_security_group_id", "flow_compute_security_group_rule.foobar_icmp", "remote_security_group_id"),
				),
			},
		},
	})
}

const testAccComputeSecurityGroupRuleConfigICMP = `
resource "flow_compute_security_group" "foobar" {
	name        = "%[1]s"
	location_id = 1
}

resource "flow_compute_security_group_rule" "foobar_icmp" {
	security_group_id = flow_compute_security_group.foobar.id

	direction = "ingress"
	protocol  = { name = "icmp" }

	icmp = {
		type = %[2]d
		code = %[3]d
	}

	ip_range = "0.0.0.0/0"
}
`

const testAccComputeSecurityGroupRuleConfigBasic = `
resource "flow_compute_security_group" "foobar" {
	name        = "%s"
	location_id = 1
}

resource "flow_compute_security_group_rule" "%s" {
	security_group_id = flow_compute_security_group.foobar.id

	direction = "%s"
	protocol  = { name = "%s" }

	port_range = {
		from = %d
		to   = %d
	}

	ip_range = "%s"
}
`

const testAccComputeSecurityGroupRuleConfigDataSource = `
data "flow_compute_security_group_rule" "by_id" {
	id                = flow_compute_security_group_rule.%[1]s.id
	security_group_id = flow_compute_security_group_rule.%[1]s.security_group_id
}
`
