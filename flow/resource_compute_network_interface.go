package flow

import (
	"context"
	"errors"
	"fmt"

	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/compute"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/flowswiss/terraform-provider-flow/filter"
)

var (
	_ tfsdk.ResourceType            = (*computeNetworkInterfaceResourceType)(nil)
	_ tfsdk.Resource                = (*computeNetworkInterfaceResource)(nil)
	_ tfsdk.ResourceWithImportState = (*computeNetworkInterfaceResource)(nil)
)

type computeNetworkInterfaceResourceData struct {
	ID        types.Int64 `tfsdk:"id"`
	ServerID  types.Int64 `tfsdk:"server_id"`
	NetworkID types.Int64 `tfsdk:"network_id"`

	PrivateIP  types.String `tfsdk:"private_ip"`
	MacAddress types.String `tfsdk:"mac_address"`

	SecurityGroupIDs types.Set  `tfsdk:"security_group_ids"`
	Security         types.Bool `tfsdk:"security"`
}

func (c *computeNetworkInterfaceResourceData) FromEntity(serverID int, iface compute.NetworkInterface) {
	c.ID = types.Int64{Value: int64(iface.ID)}
	c.ServerID = types.Int64{Value: int64(serverID)}
	c.NetworkID = types.Int64{Value: int64(iface.Network.ID)}

	c.PrivateIP = types.String{Value: iface.PrivateIP}
	c.MacAddress = types.String{Value: iface.MacAddress}

	c.SecurityGroupIDs = securityGroupIDSet(iface)
	c.Security = types.Bool{Value: iface.Security}
}

func (c computeNetworkInterfaceResourceData) AppliesTo(iface compute.NetworkInterface) bool {
	return c.ID.Value == int64(iface.ID)
}

type computeNetworkInterfaceResourceType struct{}

func (c computeNetworkInterfaceResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		MarkdownDescription: "Import: `terraform import flow_compute_network_interface.<name> <server_id>:<id>`",
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
					tfsdk.UseStateForUnknown(),
				},
			},
			"mac_address": {
				Type:                types.StringType,
				MarkdownDescription: "MAC address of the network interface",
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},

			"security_group_ids": {
				Type:                types.SetType{ElemType: types.Int64Type},
				MarkdownDescription: "security groups attached to the network interface — the organisation's default group when omitted; at least one is required while `security` is enabled, set `security = false` to detach all",
				Optional:            true,
				Computed:            true,
			},
			"security": {
				Type:                types.BoolType,
				MarkdownDescription: "whether to enable security groups on the network interface — enabled by default; enabling it resets the groups to the organisation's default group, disabling it detaches all groups",
				Optional:            true,
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
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

	var iface compute.NetworkInterface
	err := retryCreate(ctx, "create network interface", func() (err error) {
		iface, err = service.Create(ctx, create)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create network interface: %s", err))
		return
	}

	ifaceID := iface.ID

	rollback := func(what string, err error) {
		_ = retryDelete(ctx, "delete network interface", func() error { return service.Delete(ctx, ifaceID) })
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("%s: %s", what, err))
	}

	if !config.Security.Null && !config.Security.Unknown && config.Security.Value != iface.Security {
		update := compute.NetworkInterfaceSecurityUpdate{Security: config.Security.Value}

		err = retry(ctx, "update network interface security", func() error {
			updated, err := service.UpdateSecurity(ctx, ifaceID, update)
			if err != nil {
				return err
			}
			iface = updated
			return nil
		})
		if err != nil {
			rollback("unable to update network interface security", err)
			return
		}
	}

	if !config.SecurityGroupIDs.Null && !config.SecurityGroupIDs.Unknown && !config.SecurityGroupIDs.Equal(securityGroupIDSet(iface)) {
		update := compute.NetworkInterfaceSecurityGroupUpdate{SecurityGroupIDs: securityGroupIDs(config.SecurityGroupIDs)}

		err = retry(ctx, "update security groups", func() error {
			updated, err := service.UpdateSecurityGroups(ctx, ifaceID, update)
			if err != nil {
				return err
			}
			iface = updated
			return nil
		})
		if err != nil {
			rollback("unable to update security groups", err)
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
		if isNotFound(err) {
			removeGone(ctx, response, fmt.Sprintf("server %d", serverID))
			return
		}
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list network interfaces: %s", err))
		return
	}

	iface, err := filter.FindOne(state, list.Items)
	if err != nil {
		if errors.Is(err, filter.ErrNoResults) {
			removeGone(ctx, response, fmt.Sprintf("network interface %d", state.ID.Value))
			return
		}
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to find network interface: %s", err))
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

	var plan computeNetworkInterfaceResourceData
	diagnostics = request.Plan.Get(ctx, &plan)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	serverID := int(state.ServerID.Value)
	ifaceID := int(state.ID.Value)
	service := c.serverService.NetworkInterfaces(serverID)

	if !plan.Security.Unknown && plan.Security.Value != state.Security.Value {
		update := compute.NetworkInterfaceSecurityUpdate{Security: plan.Security.Value}

		var iface compute.NetworkInterface
		err := retry(ctx, "update network interface security", func() (err error) {
			iface, err = service.UpdateSecurity(ctx, ifaceID, update)
			return err
		})
		if err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update network interface security: %s", err))
			return
		}

		state.FromEntity(serverID, iface)
	}

	if !plan.SecurityGroupIDs.Unknown && !plan.SecurityGroupIDs.Null && !plan.SecurityGroupIDs.Equal(state.SecurityGroupIDs) {
		update := compute.NetworkInterfaceSecurityGroupUpdate{SecurityGroupIDs: securityGroupIDs(plan.SecurityGroupIDs)}

		var iface compute.NetworkInterface
		err := retry(ctx, "update security groups", func() (err error) {
			iface, err = service.UpdateSecurityGroups(ctx, ifaceID, update)
			return err
		})
		if err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update security groups: %s", err))
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

	err := retryDelete(ctx, "delete network interface", func() error {
		return c.serverService.NetworkInterfaces(serverID).Delete(ctx, ifaceID)
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete network interface: %s", err))
		return
	}
}

func securityGroupIDSet(iface compute.NetworkInterface) types.Set {
	elems := make([]attr.Value, len(iface.SecurityGroups))
	for i, group := range iface.SecurityGroups {
		elems[i] = types.Int64{Value: int64(group.ID)}
	}
	return types.Set{ElemType: types.Int64Type, Elems: elems}
}

func securityGroupIDs(set types.Set) []int {
	ids := make([]int, 0, len(set.Elems))
	for _, elem := range set.Elems {
		if id, ok := elem.(types.Int64); ok && !id.Null && !id.Unknown {
			ids = append(ids, int(id.Value))
		}
	}
	return ids
}

func (c computeNetworkInterfaceResource) ImportState(ctx context.Context, request tfsdk.ImportResourceStateRequest, response *tfsdk.ImportResourceStateResponse) {
	importStateCompositeInt64IDs(ctx, request, response, path.Root("server_id"), path.Root("id"))
}
