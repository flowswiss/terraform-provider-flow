package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/compute"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/flowswiss/terraform-provider-flow/validators"
)

var (
	_ tfsdk.ResourceType                 = (*computeSecurityGroupRuleResourceType)(nil)
	_ tfsdk.Resource                     = (*computeSecurityGroupRuleResource)(nil)
	_ tfsdk.ResourceWithConfigValidators = (*computeSecurityGroupRuleResource)(nil)
)

var protocolNumberToName = map[int]string{
	compute.ProtocolAny:  "any",
	compute.ProtocolICMP: "icmp",
	compute.ProtocolUDP:  "udp",
	compute.ProtocolTCP:  "tcp",
}

var protocolNamesToNumber = map[string]int{
	"any":  compute.ProtocolAny,
	"icmp": compute.ProtocolICMP,
	"udp":  compute.ProtocolUDP,
	"tcp":  compute.ProtocolTCP,
}

type computeSecurityGroupRuleResourceProtocol struct {
	Number types.Int64  `tfsdk:"number"`
	Name   types.String `tfsdk:"name"`
}

func (c *computeSecurityGroupRuleResourceProtocol) FromNumber(number int) {
	c.Number = types.Int64{Value: int64(number)}

	name, found := protocolNumberToName[number]
	c.Name = types.String{Value: name, Null: !found}
}

func (c computeSecurityGroupRuleResourceProtocol) ToNumber() int {
	if !c.Number.Null {
		return int(c.Number.Value)
	}

	if !c.Name.Null {
		return protocolNamesToNumber[c.Name.Value]
	}

	return 0
}

type computeSecurityGroupRuleResourcePortRange struct {
	From types.Int64 `tfsdk:"from"`
	To   types.Int64 `tfsdk:"to"`
}

type computeSecurityGroupRuleResourceICMP struct {
	Type types.Int64 `tfsdk:"type"`
	Code types.Int64 `tfsdk:"code"`
}

type computeSecurityGroupRuleResourceData struct {
	ID              types.Int64 `tfsdk:"id"`
	SecurityGroupID types.Int64 `tfsdk:"security_group_id"`

	Direction types.String                              `tfsdk:"direction"`
	Protocol  *computeSecurityGroupRuleResourceProtocol `tfsdk:"protocol"`

	PortRange *computeSecurityGroupRuleResourcePortRange `tfsdk:"port_range"`
	ICMP      *computeSecurityGroupRuleResourceICMP      `tfsdk:"icmp"`

	IPRange               types.String `tfsdk:"ip_range"`
	RemoteSecurityGroupID types.Int64  `tfsdk:"remote_security_group_id"`
}

func (c *computeSecurityGroupRuleResourceData) FromEntity(securityGroupID int, rule compute.SecurityGroupRule) {
	c.ID = types.Int64{Value: int64(rule.ID)}
	c.SecurityGroupID = types.Int64{Value: int64(securityGroupID)}

	c.Direction = types.String{Value: rule.Direction}
	c.Protocol = &computeSecurityGroupRuleResourceProtocol{}
	c.Protocol.FromNumber(rule.Protocol)

	if rule.Protocol == compute.ProtocolTCP || rule.Protocol == compute.ProtocolUDP {
		c.PortRange = &computeSecurityGroupRuleResourcePortRange{
			From: types.Int64{Value: int64(rule.FromPort)},
			To:   types.Int64{Value: int64(rule.ToPort)},
		}
	}

	if rule.Protocol == compute.ProtocolICMP {
		c.ICMP = &computeSecurityGroupRuleResourceICMP{
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

type computeSecurityGroupRuleResourceType struct{}

func (c computeSecurityGroupRuleResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the security group rule",
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},
			"security_group_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the security group",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"direction": {
				Type:                types.StringType,
				MarkdownDescription: "direction of the security group rule (ingress or egress)",
				Required:            true,
			},
			"protocol": {
				Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
					"number": {
						Type:                types.Int64Type,
						MarkdownDescription: "iana protocol number of the security group rule",
						Optional:            true,
						Computed:            true,
						PlanModifiers: tfsdk.AttributePlanModifiers{
							tfsdk.UseStateForUnknown(),
						},
					},
					"name": {
						Type:                types.StringType,
						MarkdownDescription: "protocol name of the security group rule",
						Optional:            true,
						Computed:            true,
						PlanModifiers: tfsdk.AttributePlanModifiers{
							tfsdk.UseStateForUnknown(),
						},
					},
				}),
				MarkdownDescription: "protocol of the security group rule",
				Required:            true,
			},
			"port_range": {
				Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
					"from": {
						Type:                types.Int64Type,
						MarkdownDescription: "starting port of the security group rule",
						Required:            true,
					},
					"to": {
						Type:                types.Int64Type,
						MarkdownDescription: "ending port of the security group rule",
						Required:            true,
					},
				}),
				MarkdownDescription: "port range filter of the security group rule",
				Optional:            true,
			},
			"icmp": {
				Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
					"type": {
						Type:                types.Int64Type,
						MarkdownDescription: "type of the ICMP message",
						Required:            true,
					},
					"code": {
						Type:                types.Int64Type,
						MarkdownDescription: "code of the ICMP message",
						Required:            true,
					},
				}),
				MarkdownDescription: "ICMP message filter of the security group rule",
				Optional:            true,
			},
			"ip_range": {
				Type:                types.StringType,
				MarkdownDescription: "ip range of the security group rule",
				Optional:            true,
			},
			"remote_security_group_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the remote security group",
				Optional:            true,
			},
		},
	}, nil
}

func (c computeSecurityGroupRuleResourceType) NewResource(ctx context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return computeSecurityGroupRuleResource{
		securityGroupService: compute.NewSecurityGroupService(prov.client),
	}, diagnostics
}

type computeSecurityGroupRuleResource struct {
	securityGroupService compute.SecurityGroupService
}

func (c computeSecurityGroupRuleResource) Create(ctx context.Context, request tfsdk.CreateResourceRequest, response *tfsdk.CreateResourceResponse) {
	var config computeSecurityGroupRuleResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	securityGroupID := int(config.SecurityGroupID.Value)
	create := compute.SecurityGroupRuleOptions{
		Direction:             config.Direction.Value,
		Protocol:              config.Protocol.ToNumber(),
		IPRange:               config.IPRange.Value,
		RemoteSecurityGroupID: int(config.RemoteSecurityGroupID.Value),
	}

	if config.PortRange != nil {
		create.FromPort = int(config.PortRange.From.Value)
		create.ToPort = int(config.PortRange.To.Value)
	}

	if config.ICMP != nil {
		create.ICMPType = int(config.ICMP.Type.Value)
		create.ICMPCode = int(config.ICMP.Code.Value)
	}

	rule, err := c.securityGroupService.Rules(securityGroupID).Create(ctx, create)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create security group rule: %s", err))
		return
	}

	var state computeSecurityGroupRuleResourceData
	state.FromEntity(securityGroupID, rule)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeSecurityGroupRuleResource) Read(ctx context.Context, request tfsdk.ReadResourceRequest, response *tfsdk.ReadResourceResponse) {
	var state computeSecurityGroupRuleResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	securityGroupID := int(state.SecurityGroupID.Value)
	ruleID := int(state.ID.Value)

	list, err := c.securityGroupService.Rules(securityGroupID).List(ctx, goclient.Cursor{NoFilter: 1})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list security group rules: %s", err))
		return
	}

	for _, rule := range list.Items {
		if rule.ID == ruleID {
			state.FromEntity(securityGroupID, rule)

			diagnostics = response.State.Set(ctx, state)
			response.Diagnostics.Append(diagnostics...)
			return
		}
	}

	response.Diagnostics.AddError("Not Found", fmt.Sprintf("security group rule %d could not be found", ruleID))
}

func (c computeSecurityGroupRuleResource) Update(ctx context.Context, request tfsdk.UpdateResourceRequest, response *tfsdk.UpdateResourceResponse) {
	var state computeSecurityGroupRuleResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	var config computeSecurityGroupRuleResourceData
	diagnostics = request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	securityGroupID := int(config.SecurityGroupID.Value)
	ruleID := int(state.ID.Value)

	update := compute.SecurityGroupRuleOptions{
		Direction:             config.Direction.Value,
		Protocol:              config.Protocol.ToNumber(),
		IPRange:               config.IPRange.Value,
		RemoteSecurityGroupID: int(config.RemoteSecurityGroupID.Value),
	}

	if config.PortRange != nil {
		update.FromPort = int(config.PortRange.From.Value)
		update.ToPort = int(config.PortRange.To.Value)
	}

	if config.ICMP != nil {
		update.ICMPType = int(config.ICMP.Type.Value)
		update.ICMPCode = int(config.ICMP.Code.Value)
	}

	rule, err := c.securityGroupService.Rules(securityGroupID).Update(ctx, ruleID, update)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update security group rule: %s", err))
		return
	}

	state.FromEntity(securityGroupID, rule)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeSecurityGroupRuleResource) Delete(ctx context.Context, request tfsdk.DeleteResourceRequest, response *tfsdk.DeleteResourceResponse) {
	var state computeSecurityGroupRuleResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	securityGroupID := int(state.SecurityGroupID.Value)
	ruleID := int(state.ID.Value)

	err := c.securityGroupService.Rules(securityGroupID).Delete(ctx, ruleID)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete security group rule: %s", err))
		return
	}
}

func (c computeSecurityGroupRuleResource) ConfigValidators(ctx context.Context) []tfsdk.ResourceConfigValidator {
	return []tfsdk.ResourceConfigValidator{
		validators.MutuallyExclusive("port_range", "icmp"),
		validators.MutuallyExclusive("ip_range", "remote_security_group_id"),
	}
}
