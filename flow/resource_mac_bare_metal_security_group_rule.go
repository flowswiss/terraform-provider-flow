package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/macbaremetal"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/flowswiss/terraform-provider-flow/validators"
)

var (
	_ tfsdk.ResourceType                 = (*macBareMetalSecurityGroupRuleResourceType)(nil)
	_ tfsdk.Resource                     = (*macBareMetalSecurityGroupRuleResource)(nil)
	_ tfsdk.ResourceWithConfigValidators = (*macBareMetalSecurityGroupRuleResource)(nil)
)

var macBareMetalProtocolNumberToName = map[int]string{
	macbaremetal.ProtocolAny:  "any",
	macbaremetal.ProtocolICMP: "icmp",
	macbaremetal.ProtocolUDP:  "udp",
	macbaremetal.ProtocolTCP:  "tcp",
}

var macBareMetalProtocolNamesToNumber = map[string]int{
	"any":  macbaremetal.ProtocolAny,
	"icmp": macbaremetal.ProtocolICMP,
	"udp":  macbaremetal.ProtocolUDP,
	"tcp":  macbaremetal.ProtocolTCP,
}

type macBareMetalSecurityGroupRuleResourceProtocol struct {
	Number types.Int64  `tfsdk:"number"`
	Name   types.String `tfsdk:"name"`
}

func (r *macBareMetalSecurityGroupRuleResourceProtocol) FromNumber(number int) {
	r.Number = types.Int64{Value: int64(number)}

	name, found := macBareMetalProtocolNumberToName[number]
	r.Name = types.String{Value: name, Null: !found}
}

func (r macBareMetalSecurityGroupRuleResourceProtocol) ToNumber() int {
	if !r.Number.Null {
		return int(r.Number.Value)
	}

	if !r.Name.Null {
		return macBareMetalProtocolNamesToNumber[r.Name.Value]
	}

	return 0
}

type macBareMetalSecurityGroupRuleResourcePortRange struct {
	From types.Int64 `tfsdk:"from"`
	To   types.Int64 `tfsdk:"to"`
}

type macBareMetalSecurityGroupRuleResourceICMP struct {
	Type types.Int64 `tfsdk:"type"`
	Code types.Int64 `tfsdk:"code"`
}

type macBareMetalSecurityGroupRuleResourceData struct {
	ID              types.Int64 `tfsdk:"id"`
	SecurityGroupID types.Int64 `tfsdk:"security_group_id"`

	Direction types.String                                   `tfsdk:"direction"`
	Protocol  *macBareMetalSecurityGroupRuleResourceProtocol `tfsdk:"protocol"`

	PortRange *macBareMetalSecurityGroupRuleResourcePortRange `tfsdk:"port_range"`
	ICMP      *macBareMetalSecurityGroupRuleResourceICMP      `tfsdk:"icmp"`

	IPRange types.String `tfsdk:"ip_range"`
}

func (r *macBareMetalSecurityGroupRuleResourceData) FromEntity(securityGroupID int, rule macbaremetal.SecurityGroupRule) {
	r.ID = types.Int64{Value: int64(rule.ID)}
	r.SecurityGroupID = types.Int64{Value: int64(securityGroupID)}

	r.Direction = types.String{Value: rule.Direction}
	r.Protocol = &macBareMetalSecurityGroupRuleResourceProtocol{}
	r.Protocol.FromNumber(rule.Protocol)

	if rule.Protocol == macbaremetal.ProtocolTCP || rule.Protocol == macbaremetal.ProtocolUDP {
		r.PortRange = &macBareMetalSecurityGroupRuleResourcePortRange{
			From: types.Int64{Value: int64(rule.FromPort)},
			To:   types.Int64{Value: int64(rule.ToPort)},
		}
	}

	if rule.Protocol == macbaremetal.ProtocolICMP {
		r.ICMP = &macBareMetalSecurityGroupRuleResourceICMP{
			Type: types.Int64{Value: int64(rule.ICMPType)},
			Code: types.Int64{Value: int64(rule.ICMPCode)},
		}
	}

	if rule.IPRange == "" {
		r.IPRange = types.String{Null: true}
	} else {
		r.IPRange = types.String{Value: rule.IPRange}
	}
}

type macBareMetalSecurityGroupRuleResourceType struct{}

func (r macBareMetalSecurityGroupRuleResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
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
		},
	}, nil
}

func (r macBareMetalSecurityGroupRuleResourceType) NewResource(ctx context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return macBareMetalSecurityGroupRuleResource{
		securityGroupService: macbaremetal.NewSecurityGroupService(prov.client),
	}, diagnostics
}

type macBareMetalSecurityGroupRuleResource struct {
	securityGroupService macbaremetal.SecurityGroupService
}

func (r macBareMetalSecurityGroupRuleResource) Create(ctx context.Context, request tfsdk.CreateResourceRequest, response *tfsdk.CreateResourceResponse) {
	var config macBareMetalSecurityGroupRuleResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	securityGroupID := int(config.SecurityGroupID.Value)
	create := macbaremetal.SecurityGroupRuleOptions{
		Direction: config.Direction.Value,
		Protocol:  config.Protocol.ToNumber(),
		IPRange:   config.IPRange.Value,
	}

	if config.PortRange != nil {
		create.FromPort = int(config.PortRange.From.Value)
		create.ToPort = int(config.PortRange.To.Value)
	}

	if config.ICMP != nil {
		create.ICMPType = int(config.ICMP.Type.Value)
		create.ICMPCode = int(config.ICMP.Code.Value)
	}

	rule, err := r.securityGroupService.Rules(securityGroupID).Create(ctx, create)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create security group rule: %s", err))
		return
	}

	var state macBareMetalSecurityGroupRuleResourceData
	state.FromEntity(securityGroupID, rule)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (r macBareMetalSecurityGroupRuleResource) Read(ctx context.Context, request tfsdk.ReadResourceRequest, response *tfsdk.ReadResourceResponse) {
	var state macBareMetalSecurityGroupRuleResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	securityGroupID := int(state.SecurityGroupID.Value)
	ruleID := int(state.ID.Value)

	list, err := r.securityGroupService.Rules(securityGroupID).List(ctx, goclient.Cursor{NoFilter: 1})
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

func (r macBareMetalSecurityGroupRuleResource) Update(ctx context.Context, request tfsdk.UpdateResourceRequest, response *tfsdk.UpdateResourceResponse) {
	var state macBareMetalSecurityGroupRuleResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	var config macBareMetalSecurityGroupRuleResourceData
	diagnostics = request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	securityGroupID := int(config.SecurityGroupID.Value)
	ruleID := int(state.ID.Value)

	update := macbaremetal.SecurityGroupRuleOptions{
		Direction: config.Direction.Value,
		Protocol:  config.Protocol.ToNumber(),
		IPRange:   config.IPRange.Value,
	}

	if config.PortRange != nil {
		update.FromPort = int(config.PortRange.From.Value)
		update.ToPort = int(config.PortRange.To.Value)
	}

	if config.ICMP != nil {
		update.ICMPType = int(config.ICMP.Type.Value)
		update.ICMPCode = int(config.ICMP.Code.Value)
	}

	rule, err := r.securityGroupService.Rules(securityGroupID).Update(ctx, ruleID, update)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update security group rule: %s", err))
		return
	}

	state.FromEntity(securityGroupID, rule)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (r macBareMetalSecurityGroupRuleResource) Delete(ctx context.Context, request tfsdk.DeleteResourceRequest, response *tfsdk.DeleteResourceResponse) {
	var state macBareMetalSecurityGroupRuleResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	securityGroupID := int(state.SecurityGroupID.Value)
	ruleID := int(state.ID.Value)

	err := r.securityGroupService.Rules(securityGroupID).Delete(ctx, ruleID)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete security group rule: %s", err))
		return
	}
}

func (r macBareMetalSecurityGroupRuleResource) ConfigValidators(ctx context.Context) []tfsdk.ResourceConfigValidator {
	return []tfsdk.ResourceConfigValidator{
		validators.MutuallyExclusive("port_range", "icmp"),
	}
}
