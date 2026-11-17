package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/v2/compute"
	"github.com/flowswiss/goclient/v2/core"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*computeSecurityGroupRuleDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*computeSecurityGroupRuleDataSource)(nil)
)

type computeSecurityGroupRuleDataSourceProtocol struct {
	Number types.Int64  `tfsdk:"number"`
	Name   types.String `tfsdk:"name"`
}

func (c *computeSecurityGroupRuleDataSourceProtocol) FromNumber(number int) {
	c.Number = types.Int64Value(int64(number))

	if name, found := protocolNumberToName[number]; found {
		c.Name = types.StringValue(name)
	} else {
		c.Name = types.StringNull()
	}
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
	c.ID = types.Int64Value(int64(rule.ID))
	c.SecurityGroupID = types.Int64Value(int64(securityGroupID))

	c.Direction = types.StringValue(rule.Direction)
	c.Protocol = &computeSecurityGroupRuleDataSourceProtocol{}
	c.Protocol.FromNumber(rule.Protocol)

	if rule.Protocol == compute.ProtocolTCP || rule.Protocol == compute.ProtocolUDP {
		c.PortRange = &computeSecurityGroupRuleDataSourcePortRange{
			From: types.Int64Value(int64(rule.FromPort)),
			To:   types.Int64Value(int64(rule.ToPort)),
		}
	}

	if rule.Protocol == compute.ProtocolICMP {
		c.ICMP = &computeSecurityGroupRuleDataSourceICMP{
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

func (c computeSecurityGroupRuleDataSourceData) AppliesTo(rule compute.SecurityGroupRule) bool {
	if !c.ID.IsNull() && c.ID.ValueInt64() != int64(rule.ID) {
		return false
	}

	return true
}

func (c computeSecurityGroupRuleDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the security group rule",
				Required:            true,
			},
			"security_group_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the security group",
				Required:            true,
			},
			"direction": schema.StringAttribute{
				MarkdownDescription: "direction of the security group rule (ingress or egress)",
				Computed:            true,
			},
			"protocol": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"number": schema.Int64Attribute{
						MarkdownDescription: "iana protocol number of the security group rule",
						Computed:            true,
					},
					"name": schema.StringAttribute{
						MarkdownDescription: "protocol name of the security group rule",
						Computed:            true,
					},
				},
				MarkdownDescription: "protocol of the security group rule",
				Computed:            true,
			},
			"port_range": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"from": schema.Int64Attribute{
						MarkdownDescription: "starting port of the security group rule",
						Computed:            true,
					},
					"to": schema.Int64Attribute{
						MarkdownDescription: "ending port of the security group rule",
						Computed:            true,
					},
				},
				MarkdownDescription: "port range of the security group rule",
				Computed:            true,
			},
			"icmp": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"type": schema.Int64Attribute{
						MarkdownDescription: "type of the ICMP message",
						Computed:            true,
					},
					"code": schema.Int64Attribute{
						MarkdownDescription: "code of the ICMP message",
						Computed:            true,
					},
				},
				MarkdownDescription: "ICMP message of the security group rule",
				Computed:            true,
			},
			"ip_range": schema.StringAttribute{
				MarkdownDescription: "ip range of the security group rule",
				Computed:            true,
			},
			"remote_security_group_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the remote security group",
				Computed:            true,
			},
		},
	}
}

func newComputeSecurityGroupRuleDataSource() datasource.DataSource {
	return &computeSecurityGroupRuleDataSource{}
}

func (c *computeSecurityGroupRuleDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_security_group_rule"
}

func (c *computeSecurityGroupRuleDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.client = client
}

type computeSecurityGroupRuleDataSource struct {
	client flowClient
}

func (c computeSecurityGroupRuleDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var config computeSecurityGroupRuleDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	securityGroupID := int(config.SecurityGroupID.ValueInt64())
	ruleID := int(config.ID.ValueInt64())

	list, err := c.client.Compute.SecurityGroupRule.List(ctx, compute.SecurityGroupRuleListReq{SecurityGroupID: uint(securityGroupID), Cursor: core.CursorAll})
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
