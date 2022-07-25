package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/compute"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/flowswiss/terraform-provider-flow/filter"
)

var (
	_ tfsdk.DataSourceType = (*computeNetworkInterfaceDataSourceType)(nil)
	_ tfsdk.DataSource     = (*computeNetworkInterfaceDataSource)(nil)
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
	c.ID = types.Int64{Value: int64(iface.ID)}
	c.ServerID = types.Int64{Value: int64(serverID)}
	c.NetworkID = types.Int64{Value: int64(iface.Network.ID)}

	c.PrivateIP = types.String{Value: iface.PrivateIP}
	c.MacAddress = types.String{Value: iface.MacAddress}

	c.SecurityGroupIDs = make([]types.Int64, len(iface.SecurityGroups))
	for idx, securityGroup := range iface.SecurityGroups {
		c.SecurityGroupIDs[idx] = types.Int64{Value: int64(securityGroup.ID)}
	}

	c.Security = types.Bool{Value: iface.Security}
}

func (c computeNetworkInterfaceDataSourceData) AppliesTo(iface compute.NetworkInterface) bool {
	if !c.ID.Null && c.ID.Value != int64(iface.ID) {
		return false
	}

	if !c.NetworkID.Null && c.NetworkID.Value != int64(iface.Network.ID) {
		return false
	}

	if !c.PrivateIP.Null && c.PrivateIP.Value != iface.PrivateIP {
		return false
	}

	if !c.MacAddress.Null && c.MacAddress.Value != iface.MacAddress {
		return false
	}

	return true
}

type computeNetworkInterfaceDataSourceType struct{}

func (c computeNetworkInterfaceDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the network interface",
				Optional:            true,
				Computed:            true,
			},
			"server_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the server",
				Required:            true,
			},
			"network_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the network",
				Optional:            true,
				Computed:            true,
			},

			"private_ip": {
				Type:                types.StringType,
				MarkdownDescription: "private IP address of the network interface",
				Optional:            true,
				Computed:            true,
			},
			"mac_address": {
				Type:                types.StringType,
				MarkdownDescription: "MAC address of the network interface",
				Optional:            true,
				Computed:            true,
			},

			"security_group_ids": {
				Type:                types.ListType{ElemType: types.Int64Type},
				MarkdownDescription: "list of security group IDs to assign to the network interface",
				Computed:            true,
			},
			"security": {
				Type:                types.BoolType,
				MarkdownDescription: "whether security groups are enabled on the network interface",
				Computed:            true,
			},
		},
	}, nil
}

func (c computeNetworkInterfaceDataSourceType) NewDataSource(ctx context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return &computeNetworkInterfaceDataSource{
		serverService: compute.NewServerService(prov.client),
	}, diagnostics
}

type computeNetworkInterfaceDataSource struct {
	serverService compute.ServerService
}

func (c computeNetworkInterfaceDataSource) Read(ctx context.Context, request tfsdk.ReadDataSourceRequest, response *tfsdk.ReadDataSourceResponse) {
	var config computeNetworkInterfaceDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	serverID := int(config.ServerID.Value)

	list, err := c.serverService.NetworkInterfaces(serverID).List(ctx, goclient.Cursor{NoFilter: 1})
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
