package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/v2/compute"
	"github.com/flowswiss/goclient/v2/core"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/flowswiss/terraform-provider-flow/validators"
)

var (
	_ resource.Resource                     = (*computeSecurityGroupRuleResource)(nil)
	_ resource.ResourceWithConfigure        = (*computeSecurityGroupRuleResource)(nil)
	_ resource.ResourceWithImportState      = (*computeSecurityGroupRuleResource)(nil)
	_ resource.ResourceWithConfigValidators = (*computeSecurityGroupRuleResource)(nil)
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
	c.Number = types.Int64Value(int64(number))

	if name, found := protocolNumberToName[number]; found {
		c.Name = types.StringValue(name)
	} else {
		c.Name = types.StringNull()
	}
}

func (c computeSecurityGroupRuleResourceProtocol) ToNumber() int {
	if !c.Number.IsNull() && !c.Number.IsUnknown() {
		return int(c.Number.ValueInt64())
	}

	if !c.Name.IsNull() && !c.Name.IsUnknown() {
		return protocolNamesToNumber[c.Name.ValueString()]
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
	c.ID = types.Int64Value(int64(rule.ID))
	c.SecurityGroupID = types.Int64Value(int64(securityGroupID))

	c.Direction = types.StringValue(rule.Direction)
	c.Protocol = &computeSecurityGroupRuleResourceProtocol{}
	c.Protocol.FromNumber(rule.Protocol)

	c.PortRange = nil
	if rule.Protocol == compute.ProtocolTCP || rule.Protocol == compute.ProtocolUDP {
		c.PortRange = &computeSecurityGroupRuleResourcePortRange{
			From: types.Int64Value(int64(rule.FromPort)),
			To:   types.Int64Value(int64(rule.ToPort)),
		}
	}

	c.ICMP = nil
	if rule.Protocol == compute.ProtocolICMP {
		c.ICMP = &computeSecurityGroupRuleResourceICMP{
			Type: types.Int64Value(int64(rule.ICMPType)),
			Code: types.Int64Value(int64(rule.ICMPCode)),
		}
	}

	if rule.IPRange == "" {
		c.IPRange = types.StringNull()
	} else {
		c.IPRange = types.StringValue(rule.IPRange)
	}

	if rule.RemoteSecurityGroup.ID == 0 {
		c.RemoteSecurityGroupID = types.Int64Null()
	} else {
		c.RemoteSecurityGroupID = types.Int64Value(int64(rule.RemoteSecurityGroup.ID))
	}
}

func (c computeSecurityGroupRuleResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Import: `terraform import flow_compute_security_group_rule.<name> <security_group_id>:<id>`",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the security group rule",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"security_group_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the security group",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"direction": schema.StringAttribute{
				MarkdownDescription: "direction of the security group rule (ingress or egress)",
				Required:            true,
			},
			"protocol": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"number": schema.Int64Attribute{
						MarkdownDescription: "iana protocol number of the security group rule",
						Optional:            true,
						Computed:            true,
					},
					"name": schema.StringAttribute{
						MarkdownDescription: "protocol name of the security group rule",
						Optional:            true,
						Computed:            true,
					},
				},
				MarkdownDescription: "protocol of the security group rule",
				Required:            true,
			},
			"port_range": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"from": schema.Int64Attribute{
						MarkdownDescription: "starting port of the security group rule",
						Required:            true,
					},
					"to": schema.Int64Attribute{
						MarkdownDescription: "ending port of the security group rule",
						Required:            true,
					},
				},
				MarkdownDescription: "port range filter of the security group rule",
				Optional:            true,
			},
			"icmp": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"type": schema.Int64Attribute{
						MarkdownDescription: "type of the ICMP message",
						Required:            true,
					},
					"code": schema.Int64Attribute{
						MarkdownDescription: "code of the ICMP message",
						Required:            true,
					},
				},
				MarkdownDescription: "ICMP message filter of the security group rule",
				Optional:            true,
			},
			"ip_range": schema.StringAttribute{
				MarkdownDescription: "ip range of the security group rule",
				Optional:            true,
			},
			"remote_security_group_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the remote security group",
				Optional:            true,
			},
		},
	}
}

func newComputeSecurityGroupRuleResource() resource.Resource {
	return &computeSecurityGroupRuleResource{}
}

func (c *computeSecurityGroupRuleResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_security_group_rule"
}

func (c *computeSecurityGroupRuleResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.client = client
}

type computeSecurityGroupRuleResource struct {
	client flowClient
}

func (c computeSecurityGroupRuleResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var config computeSecurityGroupRuleResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	securityGroupID := int(config.SecurityGroupID.ValueInt64())
	create := compute.SecurityGroupRuleCreateReq{
		SecurityGroupID:       uint(securityGroupID),
		Direction:             config.Direction.ValueString(),
		Protocol:              config.Protocol.ToNumber(),
		IPRange:               nonZero(config.IPRange.ValueString()),
		RemoteSecurityGroupID: nonZero(int(config.RemoteSecurityGroupID.ValueInt64())),
	}

	if config.PortRange != nil {
		create.FromPort = nonZero(int(config.PortRange.From.ValueInt64()))
		create.ToPort = nonZero(int(config.PortRange.To.ValueInt64()))
	}

	if config.ICMP != nil {
		create.ICMPType = intPointer(config.ICMP.Type)
		create.ICMPCode = intPointer(config.ICMP.Code)
	}

	var rule compute.SecurityGroupRule
	err := retryCreate(ctx, "create security group rule", func() (err error) {
		rule, err = c.client.Compute.SecurityGroupRule.Create(ctx, create)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create security group rule: %s", err))
		return
	}

	var state computeSecurityGroupRuleResourceData
	state.FromEntity(securityGroupID, rule)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeSecurityGroupRuleResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state computeSecurityGroupRuleResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	securityGroupID := int(state.SecurityGroupID.ValueInt64())
	ruleID := int(state.ID.ValueInt64())

	list, err := c.client.Compute.SecurityGroupRule.List(ctx, compute.SecurityGroupRuleListReq{
		SecurityGroupID: uint(securityGroupID),
		Cursor:          core.CursorAll,
	})
	if err != nil {
		if isNotFound(err) {
			removeGone(ctx, response, fmt.Sprintf("security group %d", securityGroupID))
			return
		}
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

	removeGone(ctx, response, fmt.Sprintf("security group rule %d", ruleID))
}

func (c computeSecurityGroupRuleResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
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

	securityGroupID := int(config.SecurityGroupID.ValueInt64())

	update := compute.SecurityGroupRuleUpdateReq{
		SecurityGroupID:       uint(securityGroupID),
		SecurityGroupRuleID:   uint(state.ID.ValueInt64()),
		Direction:             new(config.Direction.ValueString()),
		Protocol:              new(config.Protocol.ToNumber()),
		IPRange:               nonZero(config.IPRange.ValueString()),
		RemoteSecurityGroupID: nonZero(int(config.RemoteSecurityGroupID.ValueInt64())),
	}

	if config.PortRange != nil {
		update.FromPort = nonZero(int(config.PortRange.From.ValueInt64()))
		update.ToPort = nonZero(int(config.PortRange.To.ValueInt64()))
	}

	if config.ICMP != nil {
		update.ICMPType = intPointer(config.ICMP.Type)
		update.ICMPCode = intPointer(config.ICMP.Code)
	}

	var rule compute.SecurityGroupRule
	err := retry(ctx, "update security group rule", func() (err error) {
		rule, err = c.client.Compute.SecurityGroupRule.Update(ctx, update)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update security group rule: %s", err))
		return
	}

	state.FromEntity(securityGroupID, rule)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeSecurityGroupRuleResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state computeSecurityGroupRuleResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	err := retryDelete(ctx, "delete security group rule", func() error {
		return c.client.Compute.SecurityGroupRule.Delete(ctx, compute.SecurityGroupRuleDeleteReq{
			SecurityGroupID:     uint(state.SecurityGroupID.ValueInt64()),
			SecurityGroupRuleID: uint(state.ID.ValueInt64()),
		})
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete security group rule: %s", err))
		return
	}
}

func (c computeSecurityGroupRuleResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		validators.MutuallyExclusive("port_range", "icmp"),
		validators.MutuallyExclusive("ip_range", "remote_security_group_id"),
	}
}

func (c computeSecurityGroupRuleResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importStateCompositeInt64IDs(ctx, request, response, path.Root("security_group_id"), path.Root("id"))
}
