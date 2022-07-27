package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/macbaremetal"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ tfsdk.DataSourceType = (*macBareMetalSecurityGroupRuleDataSourceType)(nil)
	_ tfsdk.DataSource     = (*macBareMetalSecurityGroupRuleDataSource)(nil)
)

type macBareMetalSecurityGroupRuleDataSourceProtocol struct {
	Number types.Int64  `tfsdk:"number"`
	Name   types.String `tfsdk:"name"`
}

func (c *macBareMetalSecurityGroupRuleDataSourceProtocol) FromNumber(number int) {
	c.Number = types.Int64{Value: int64(number)}

	name, found := protocolNumberToName[number]
	c.Name = types.String{Value: name, Null: !found}
}

type macBareMetalSecurityGroupRuleDataSourcePortRange struct {
	From types.Int64 `tfsdk:"from"`
	To   types.Int64 `tfsdk:"to"`
}

type macBareMetalSecurityGroupRuleDataSourceICMP struct {
	Type types.Int64 `tfsdk:"type"`
	Code types.Int64 `tfsdk:"code"`
}

type macBareMetalSecurityGroupRuleDataSourceData struct {
	ID              types.Int64 `tfsdk:"id"`
	SecurityGroupID types.Int64 `tfsdk:"security_group_id"`

	Direction types.String                                     `tfsdk:"direction"`
	Protocol  *macBareMetalSecurityGroupRuleDataSourceProtocol `tfsdk:"protocol"`

	PortRange *macBareMetalSecurityGroupRuleDataSourcePortRange `tfsdk:"port_range"`
	ICMP      *macBareMetalSecurityGroupRuleDataSourceICMP      `tfsdk:"icmp"`

	IPRange types.String `tfsdk:"ip_range"`
}

func (c *macBareMetalSecurityGroupRuleDataSourceData) FromEntity(securityGroupID int, rule macbaremetal.SecurityGroupRule) {
	c.ID = types.Int64{Value: int64(rule.ID)}
	c.SecurityGroupID = types.Int64{Value: int64(securityGroupID)}

	c.Direction = types.String{Value: rule.Direction}
	c.Protocol = &macBareMetalSecurityGroupRuleDataSourceProtocol{}
	c.Protocol.FromNumber(rule.Protocol)

	if rule.FromPort != 0 && rule.ToPort != 0 {
		c.PortRange = &macBareMetalSecurityGroupRuleDataSourcePortRange{
			From: types.Int64{Value: int64(rule.FromPort)},
			To:   types.Int64{Value: int64(rule.ToPort)},
		}
	}

	if rule.ICMPType != 0 && rule.ICMPCode != 0 {
		c.ICMP = &macBareMetalSecurityGroupRuleDataSourceICMP{
			Type: types.Int64{Value: int64(rule.ICMPType)},
			Code: types.Int64{Value: int64(rule.ICMPCode)},
		}
	}

	if rule.IPRange == "" {
		c.IPRange = types.String{Null: true}
	} else {
		c.IPRange = types.String{Value: rule.IPRange}
	}
}

func (c macBareMetalSecurityGroupRuleDataSourceData) AppliesTo(rule macbaremetal.SecurityGroupRule) bool {
	if !c.ID.Null && c.ID.Value != int64(rule.ID) {
		return false
	}

	return true
}

type macBareMetalSecurityGroupRuleDataSourceType struct{}

func (c macBareMetalSecurityGroupRuleDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
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
		},
	}, nil
}

func (c macBareMetalSecurityGroupRuleDataSourceType) NewDataSource(ctx context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return macBareMetalSecurityGroupRuleDataSource{
		securityGroupService: macbaremetal.NewSecurityGroupService(prov.client),
	}, diagnostics
}

type macBareMetalSecurityGroupRuleDataSource struct {
	securityGroupService macbaremetal.SecurityGroupService
}

func (c macBareMetalSecurityGroupRuleDataSource) Read(ctx context.Context, request tfsdk.ReadDataSourceRequest, response *tfsdk.ReadDataSourceResponse) {
	var config macBareMetalSecurityGroupRuleDataSourceData
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
			var state macBareMetalSecurityGroupRuleDataSourceData
			state.FromEntity(securityGroupID, rule)

			diagnostics = response.State.Set(ctx, state)
			response.Diagnostics.Append(diagnostics...)
			return
		}
	}

	response.Diagnostics.AddError("Not Found", fmt.Sprintf("security group rule %d could not be found", ruleID))
}
