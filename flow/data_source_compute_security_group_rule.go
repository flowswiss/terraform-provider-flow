package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/compute"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ tfsdk.DataSourceType = (*computeSecurityGroupRuleDataSourceType)(nil)
	_ tfsdk.DataSource     = (*computeSecurityGroupRuleDataSource)(nil)
)

type computeSecurityGroupRuleDataSourceProtocol struct {
	Number types.Int64  `tfsdk:"number"`
	Name   types.String `tfsdk:"name"`
}

func (c *computeSecurityGroupRuleDataSourceProtocol) FromNumber(number int) {
	c.Number = types.Int64{Value: int64(number)}

	name, found := protocolNumberToName[number]
	c.Name = types.String{Value: name, Null: !found}
}

type computeSecurityGroupRuleDataSourcePortRange struct {
	From types.Int64 `tfsdk:"from"`
	To   types.Int64 `tfsdk:"to"`
}

type computeSecurityGroupRuleDataSourceICMP struct {
	Type types.Int64 `tfsdk:"type"`
	Code types.Int64 `tfsdk:"code"`
}

type computeSecurityGroupRuleDataSourceData struct {
	ID              types.Int64 `tfsdk:"id"`
	SecurityGroupID types.Int64 `tfsdk:"security_group_id"`

	Direction types.String                                `tfsdk:"direction"`
	Protocol  *computeSecurityGroupRuleDataSourceProtocol `tfsdk:"protocol"`

	PortRange *computeSecurityGroupRuleDataSourcePortRange `tfsdk:"port_range"`
	ICMP      *computeSecurityGroupRuleDataSourceICMP      `tfsdk:"icmp"`

	IPRange               types.String `tfsdk:"ip_range"`
	RemoteSecurityGroupID types.Int64  `tfsdk:"remote_security_group_id"`
}

func (c *computeSecurityGroupRuleDataSourceData) FromEntity(securityGroupID int, rule compute.SecurityGroupRule) {
	c.ID = types.Int64{Value: int64(rule.ID)}
	c.SecurityGroupID = types.Int64{Value: int64(securityGroupID)}

	c.Direction = types.String{Value: rule.Direction}
	c.Protocol = &computeSecurityGroupRuleDataSourceProtocol{}
	c.Protocol.FromNumber(rule.Protocol)

	if rule.FromPort != 0 && rule.ToPort != 0 {
		c.PortRange = &computeSecurityGroupRuleDataSourcePortRange{
			From: types.Int64{Value: int64(rule.FromPort)},
			To:   types.Int64{Value: int64(rule.ToPort)},
		}
	}

	if rule.ICMPType != 0 && rule.ICMPCode != 0 {
		c.ICMP = &computeSecurityGroupRuleDataSourceICMP{
			Type: types.Int64{Value: int64(rule.ICMPType)},
			Code: types.Int64{Value: int64(rule.ICMPCode)},
		}
	}

	if rule.IPRange == "" {
		c.IPRange = types.String{Null: true}
	} else {
		c.IPRange = types.String{Value: rule.IPRange}
	}

	if rule.RemoteSecurityGroup.ID == 0 {
		c.RemoteSecurityGroupID = types.Int64{Null: true}
	} else {
		c.RemoteSecurityGroupID = types.Int64{Value: int64(rule.RemoteSecurityGroup.ID)}
	}
}

func (c computeSecurityGroupRuleDataSourceData) AppliesTo(rule compute.SecurityGroupRule) bool {
	if !c.ID.Null && c.ID.Value != int64(rule.ID) {
		return false
	}

	return true
}

type computeSecurityGroupRuleDataSourceType struct{}

func (c computeSecurityGroupRuleDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the security group rule",
				Required:            true,
			},
			"security_group_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the security group",
				Required:            true,
			},
			"direction": {
				Type:                types.StringType,
				MarkdownDescription: "direction of the security group rule (ingress or egress)",
				Computed:            true,
			},
			"protocol": {
				Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
					"number": {
						Type:                types.Int64Type,
						MarkdownDescription: "iana protocol number of the security group rule",
						Computed:            true,
					},
					"name": {
						Type:                types.StringType,
						MarkdownDescription: "protocol name of the security group rule",
						Computed:            true,
					},
				}),
				MarkdownDescription: "protocol of the security group rule",
				Computed:            true,
			},
			"port_range": {
				Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
					"from": {
						Type:                types.Int64Type,
						MarkdownDescription: "starting port of the security group rule",
						Computed:            true,
					},
					"to": {
						Type:                types.Int64Type,
						MarkdownDescription: "ending port of the security group rule",
						Computed:            true,
					},
				}),
				MarkdownDescription: "port range of the security group rule",
				Computed:            true,
			},
			"icmp": {
				Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
					"type": {
						Type:                types.Int64Type,
						MarkdownDescription: "type of the ICMP message",
						Computed:            true,
					},
					"code": {
						Type:                types.Int64Type,
						MarkdownDescription: "code of the ICMP message",
						Computed:            true,
					},
				}),
				MarkdownDescription: "ICMP message of the security group rule",
				Computed:            true,
			},
			"ip_range": {
				Type:                types.StringType,
				MarkdownDescription: "ip range of the security group rule",
				Computed:            true,
			},
			"remote_security_group_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the remote security group",
				Computed:            true,
			},
		},
	}, nil
}

func (c computeSecurityGroupRuleDataSourceType) NewDataSource(ctx context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return computeSecurityGroupRuleDataSource{
		securityGroupService: compute.NewSecurityGroupService(prov.client),
	}, diagnostics
}

type computeSecurityGroupRuleDataSource struct {
	securityGroupService compute.SecurityGroupService
}

func (c computeSecurityGroupRuleDataSource) Read(ctx context.Context, request tfsdk.ReadDataSourceRequest, response *tfsdk.ReadDataSourceResponse) {
	var config computeSecurityGroupRuleDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	securityGroupID := int(config.SecurityGroupID.Value)
	ruleID := int(config.ID.Value)

	list, err := c.securityGroupService.Rules(securityGroupID).List(ctx, goclient.Cursor{NoFilter: 1})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list security group rules: %s", err))
		return
	}

	for _, rule := range list.Items {
		if rule.ID == ruleID {
			var state computeSecurityGroupRuleDataSourceData
			state.FromEntity(securityGroupID, rule)

			diagnostics = response.State.Set(ctx, state)
			response.Diagnostics.Append(diagnostics...)
			return
		}
	}

	response.Diagnostics.AddError("Not Found", fmt.Sprintf("security group rule %d could not be found", ruleID))
}
