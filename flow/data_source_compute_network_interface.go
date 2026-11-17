package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/v2/compute"
	"github.com/flowswiss/goclient/v2/core"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/flowswiss/terraform-provider-flow/filter"
)

var (
	_ datasource.DataSource              = (*computeNetworkInterfaceDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*computeNetworkInterfaceDataSource)(nil)
)

type computeNetworkInterfaceDataSourceData struct {
	ID        types.Int64 `tfsdk:"id"`
	ServerID  types.Int64 `tfsdk:"server_id"`
	NetworkID types.Int64 `tfsdk:"network_id"`

	PrivateIP  types.String `tfsdk:"private_ip"`
	MacAddress types.String `tfsdk:"mac_address"`

	SecurityGroupIDs []types.Int64 `tfsdk:"security_group_ids"`
	Security         types.Bool    `tfsdk:"security"`
}

func (c *computeNetworkInterfaceDataSourceData) FromEntity(serverID int, iface compute.NetworkInterface) {
	c.ID = types.Int64Value(int64(iface.ID))
	c.ServerID = types.Int64Value(int64(serverID))
	c.NetworkID = types.Int64Value(int64(iface.Network.ID))

	c.PrivateIP = types.StringValue(iface.PrivateIP)
	c.MacAddress = types.StringValue(iface.MacAddress)

	c.SecurityGroupIDs = make([]types.Int64, len(iface.SecurityGroups))
	for idx, securityGroup := range iface.SecurityGroups {
		c.SecurityGroupIDs[idx] = types.Int64Value(int64(securityGroup.ID))
	}

	c.Security = types.BoolValue(iface.Security)
}

func (c computeNetworkInterfaceDataSourceData) AppliesTo(iface compute.NetworkInterface) bool {
	if !c.ID.IsNull() && c.ID.ValueInt64() != int64(iface.ID) {
		return false
	}

	if !c.NetworkID.IsNull() && c.NetworkID.ValueInt64() != int64(iface.Network.ID) {
		return false
	}

	if !c.PrivateIP.IsNull() && c.PrivateIP.ValueString() != iface.PrivateIP {
		return false
	}

	if !c.MacAddress.IsNull() && c.MacAddress.ValueString() != iface.MacAddress {
		return false
	}

	return true
}

func (c computeNetworkInterfaceDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the network interface",
				Optional:            true,
				Computed:            true,
			},
			"server_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the server",
				Required:            true,
			},
			"network_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the network",
				Optional:            true,
				Computed:            true,
			},

			"private_ip": schema.StringAttribute{
				MarkdownDescription: "private IP address of the network interface",
				Optional:            true,
				Computed:            true,
			},
			"mac_address": schema.StringAttribute{
				MarkdownDescription: "MAC address of the network interface",
				Optional:            true,
				Computed:            true,
			},

			"security_group_ids": schema.ListAttribute{
				ElementType:         types.Int64Type,
				MarkdownDescription: "list of security group IDs to assign to the network interface",
				Computed:            true,
			},
			"security": schema.BoolAttribute{
				MarkdownDescription: "whether security groups are enabled on the network interface",
				Computed:            true,
			},
		},
	}
}

func newComputeNetworkInterfaceDataSource() datasource.DataSource {
	return &computeNetworkInterfaceDataSource{}
}

func (c *computeNetworkInterfaceDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_network_interface"
}

func (c *computeNetworkInterfaceDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.client = client
}

type computeNetworkInterfaceDataSource struct {
	client flowClient
}

func (c computeNetworkInterfaceDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var config computeNetworkInterfaceDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	serverID := int(config.ServerID.ValueInt64())

	list, err := c.client.Compute.NetworkInterface.List(ctx, compute.NetworkInterfaceListReq{ServerID: uint(serverID), Cursor: core.CursorAll})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list network interfaces: %s", err))
		return
	}

	iface, err := filter.FindOne(config, list.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find network interface: %s", err))
		return
	}

	var state computeNetworkInterfaceDataSourceData
	state.FromEntity(serverID, iface)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
