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
	_ tfsdk.ResourceType = (*computeNetworkInterfaceResourceType)(nil)
	_ tfsdk.Resource     = (*computeNetworkInterfaceResource)(nil)
)

type computeNetworkInterfaceResourceData struct {
	ID        types.Int64 `tfsdk:"id"`
	ServerID  types.Int64 `tfsdk:"server_id"`
	NetworkID types.Int64 `tfsdk:"network_id"`

	PrivateIP  types.String `tfsdk:"private_ip"`
	MacAddress types.String `tfsdk:"mac_address"`

	SecurityGroupIDs []types.Int64 `tfsdk:"security_group_ids"`
	Security         types.Bool    `tfsdk:"security"`
}

func (c *computeNetworkInterfaceResourceData) FromEntity(serverID int, iface compute.NetworkInterface) {
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

func (c computeNetworkInterfaceResourceData) AppliesTo(iface compute.NetworkInterface) bool {
	return c.ID.Value == int64(iface.ID)
}

type computeNetworkInterfaceResourceType struct{}

func (c computeNetworkInterfaceResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the network interface",
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},
			"server_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the server",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"network_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the network",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},

			"private_ip": {
				Type:                types.StringType,
				MarkdownDescription: "private IP address of the network interface",
				Optional:            true,
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"mac_address": {
				Type:                types.StringType,
				MarkdownDescription: "MAC address of the network interface",
				Computed:            true,
			},

			"security_group_ids": {
				Type:                types.ListType{ElemType: types.Int64Type},
				MarkdownDescription: "list of security group IDs to assign to the network interface",
				Optional:            true,
				Computed:            true,
			},
			"security": {
				Type:                types.BoolType,
				MarkdownDescription: "whether to enable security groups on the network interface",
				Optional:            true,
				Computed:            true,
			},
		},
	}, nil
}

func (c computeNetworkInterfaceResourceType) NewResource(ctx context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return &computeNetworkInterfaceResource{
		serverService: compute.NewServerService(prov.client),
	}, diagnostics
}

type computeNetworkInterfaceResource struct {
	serverService compute.ServerService
}

func (c computeNetworkInterfaceResource) Create(ctx context.Context, request tfsdk.CreateResourceRequest, response *tfsdk.CreateResourceResponse) {
	var config computeNetworkInterfaceResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	serverID := int(config.ServerID.Value)
	service := c.serverService.NetworkInterfaces(serverID)

	create := compute.NetworkInterfaceCreate{
		NetworkID: int(config.NetworkID.Value),
		PrivateIP: config.PrivateIP.Value,
	}

	iface, err := service.Create(ctx, create)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create network interface: %s", err))
		return
	}

	if len(config.SecurityGroupIDs) != 0 {
		update := compute.NetworkInterfaceSecurityGroupUpdate{
			SecurityGroupIDs: make([]int, len(config.SecurityGroupIDs)),
		}

		for idx, securityGroupID := range config.SecurityGroupIDs {
			update.SecurityGroupIDs[idx] = int(securityGroupID.Value)
		}

		iface, err = service.UpdateSecurityGroups(ctx, iface.ID, update)
		if err != nil {
			// delete the interface if we failed to update the security groups
			// TODO: should we add a backoff here if the deletion fails?
			_ = service.Delete(ctx, iface.ID)

			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update security groups: %s", err))
			return
		}
	}

	if !config.Security.Null && !config.Security.Value {
		update := compute.NetworkInterfaceSecurityUpdate{
			Security: config.Security.Value,
		}

		iface, err = service.UpdateSecurity(ctx, iface.ID, update)
		if err != nil {
			// delete the interface if we failed to update the security
			// TODO: should we add a backoff here if the deletion fails?
			_ = service.Delete(ctx, iface.ID)

			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update network interface security: %s", err))
			return
		}
	}

	var state computeNetworkInterfaceResourceData
	state.FromEntity(serverID, iface)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeNetworkInterfaceResource) Read(ctx context.Context, request tfsdk.ReadResourceRequest, response *tfsdk.ReadResourceResponse) {
	var state computeNetworkInterfaceResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	serverID := int(state.ServerID.Value)

	list, err := c.serverService.NetworkInterfaces(serverID).List(ctx, goclient.Cursor{NoFilter: 1})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list network interfaces: %s", err))
		return
	}

	iface, err := filter.FindOne(state, list.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find network interface: %s", err))
		return
	}

	state.FromEntity(serverID, iface)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeNetworkInterfaceResource) Update(ctx context.Context, request tfsdk.UpdateResourceRequest, response *tfsdk.UpdateResourceResponse) {
	var state computeNetworkInterfaceResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	var config computeNetworkInterfaceResourceData
	diagnostics = request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	serverID := int(state.ServerID.Value)
	ifaceID := int(state.ID.Value)

	service := c.serverService.NetworkInterfaces(serverID)

	if len(config.SecurityGroupIDs) != 0 {
		update := compute.NetworkInterfaceSecurityGroupUpdate{
			SecurityGroupIDs: make([]int, len(config.SecurityGroupIDs)),
		}

		for idx, securityGroupID := range config.SecurityGroupIDs {
			update.SecurityGroupIDs[idx] = int(securityGroupID.Value)
		}

		iface, err := service.UpdateSecurityGroups(ctx, ifaceID, update)
		if err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update security groups: %s", err))
			return
		}

		state.FromEntity(serverID, iface)
	}

	if !config.Security.Null && config.Security.Value != state.Security.Value {
		update := compute.NetworkInterfaceSecurityUpdate{
			Security: config.Security.Value,
		}

		iface, err := service.UpdateSecurity(ctx, ifaceID, update)
		if err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update network interface security: %s", err))
			return
		}

		state.FromEntity(serverID, iface)
	}

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeNetworkInterfaceResource) Delete(ctx context.Context, request tfsdk.DeleteResourceRequest, response *tfsdk.DeleteResourceResponse) {
	var state computeNetworkInterfaceResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	serverID := int(state.ServerID.Value)
	ifaceID := int(state.ID.Value)

	err := c.serverService.NetworkInterfaces(serverID).Delete(ctx, ifaceID)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete network interface: %s", err))
		return
	}
}
