package flow

import (
	"context"
	"errors"
	"fmt"

	"github.com/flowswiss/goclient/v2/compute"
	"github.com/flowswiss/goclient/v2/core"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/flowswiss/terraform-provider-flow/filter"
)

var (
	_ resource.Resource                = (*computeNetworkInterfaceResource)(nil)
	_ resource.ResourceWithConfigure   = (*computeNetworkInterfaceResource)(nil)
	_ resource.ResourceWithImportState = (*computeNetworkInterfaceResource)(nil)
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
	c.ID = types.Int64Value(int64(iface.ID))
	c.ServerID = types.Int64Value(int64(serverID))
	c.NetworkID = types.Int64Value(int64(iface.Network.ID))

	c.PrivateIP = types.StringValue(iface.PrivateIP)
	c.MacAddress = types.StringValue(iface.MacAddress)

	c.SecurityGroupIDs = securityGroupIDSet(iface)
	c.Security = types.BoolValue(iface.Security)
}

func (c computeNetworkInterfaceResourceData) AppliesTo(iface compute.NetworkInterface) bool {
	return c.ID.ValueInt64() == int64(iface.ID)
}

func (c computeNetworkInterfaceResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Import: `terraform import flow_compute_network_interface.<name> <server_id>:<id>`",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the network interface",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"server_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the server",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"network_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the network",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},

			"private_ip": schema.StringAttribute{
				MarkdownDescription: "private IP address of the network interface",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"mac_address": schema.StringAttribute{
				MarkdownDescription: "MAC address of the network interface",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"security_group_ids": schema.SetAttribute{
				ElementType:         types.Int64Type,
				MarkdownDescription: "security groups attached to the network interface — the organisation's default group when omitted; at least one is required while `security` is enabled, set `security = false` to detach all",
				Optional:            true,
				Computed:            true,
			},
			"security": schema.BoolAttribute{
				MarkdownDescription: "whether to enable security groups on the network interface — enabled by default; enabling it resets the groups to the organisation's default group, disabling it detaches all groups",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func newComputeNetworkInterfaceResource() resource.Resource {
	return &computeNetworkInterfaceResource{}
}

func (c *computeNetworkInterfaceResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_network_interface"
}

func (c *computeNetworkInterfaceResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.client = client
}

type computeNetworkInterfaceResource struct {
	client flowClient
}

func (c computeNetworkInterfaceResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var config computeNetworkInterfaceResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	serverID := int(config.ServerID.ValueInt64())
	service := c.client.Compute.NetworkInterface

	create := compute.NetworkInterfaceCreateReq{
		ServerID:  uint(serverID),
		NetworkID: int(config.NetworkID.ValueInt64()),
		PrivateIP: config.PrivateIP.ValueString(),
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
		_ = retryDelete(ctx, "delete network interface", func() error {
			return service.Delete(ctx, compute.NetworkInterfaceDeleteReq{ServerID: uint(serverID), NetworkInterfaceID: uint(ifaceID)})
		})
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("%s: %s", what, err))
	}

	if !config.Security.IsNull() && !config.Security.IsUnknown() && config.Security.ValueBool() != iface.Security {
		update := compute.NetworkInterfaceSecurityUpdateReq{
			ServerID:           uint(serverID),
			NetworkInterfaceID: uint(ifaceID),
			Security:           config.Security.ValueBool(),
		}

		err = retry(ctx, "update network interface security", func() error {
			updated, err := service.UpdateSecurity(ctx, update)
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

	if !config.SecurityGroupIDs.IsNull() && !config.SecurityGroupIDs.IsUnknown() && !config.SecurityGroupIDs.Equal(securityGroupIDSet(iface)) {
		update := compute.NetworkInterfaceSecurityGroupUpdateReq{
			ServerID:           uint(serverID),
			NetworkInterfaceID: uint(ifaceID),
			SecurityGroupIDs:   securityGroupIDs(config.SecurityGroupIDs),
		}

		err = retry(ctx, "update security groups", func() error {
			updated, err := service.UpdateSecurityGroups(ctx, update)
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

func (c computeNetworkInterfaceResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state computeNetworkInterfaceResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	serverID := int(state.ServerID.ValueInt64())

	list, err := c.client.Compute.NetworkInterface.List(ctx, compute.NetworkInterfaceListReq{ServerID: uint(serverID), Cursor: core.CursorAll})
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
			removeGone(ctx, response, fmt.Sprintf("network interface %d", state.ID.ValueInt64()))
			return
		}
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to find network interface: %s", err))
		return
	}

	state.FromEntity(serverID, iface)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeNetworkInterfaceResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
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

	serverID := int(state.ServerID.ValueInt64())
	ifaceID := int(state.ID.ValueInt64())
	service := c.client.Compute.NetworkInterface

	if !plan.Security.IsUnknown() && plan.Security.ValueBool() != state.Security.ValueBool() {
		update := compute.NetworkInterfaceSecurityUpdateReq{
			ServerID:           uint(serverID),
			NetworkInterfaceID: uint(ifaceID),
			Security:           plan.Security.ValueBool(),
		}

		var iface compute.NetworkInterface
		err := retry(ctx, "update network interface security", func() (err error) {
			iface, err = service.UpdateSecurity(ctx, update)
			return err
		})
		if err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update network interface security: %s", err))
			return
		}

		state.FromEntity(serverID, iface)
	}

	if !plan.SecurityGroupIDs.IsUnknown() && !plan.SecurityGroupIDs.IsNull() && !plan.SecurityGroupIDs.Equal(state.SecurityGroupIDs) {
		update := compute.NetworkInterfaceSecurityGroupUpdateReq{
			ServerID:           uint(serverID),
			NetworkInterfaceID: uint(ifaceID),
			SecurityGroupIDs:   securityGroupIDs(plan.SecurityGroupIDs),
		}

		var iface compute.NetworkInterface
		err := retry(ctx, "update security groups", func() (err error) {
			iface, err = service.UpdateSecurityGroups(ctx, update)
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

func (c computeNetworkInterfaceResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state computeNetworkInterfaceResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	serverID := int(state.ServerID.ValueInt64())
	ifaceID := int(state.ID.ValueInt64())

	err := retryDelete(ctx, "delete network interface", func() error {
		return c.client.Compute.NetworkInterface.Delete(ctx, compute.NetworkInterfaceDeleteReq{ServerID: uint(serverID), NetworkInterfaceID: uint(ifaceID)})
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete network interface: %s", err))
		return
	}
}

func securityGroupIDSet(iface compute.NetworkInterface) types.Set {
	elems := make([]attr.Value, len(iface.SecurityGroups))
	for i, group := range iface.SecurityGroups {
		elems[i] = types.Int64Value(int64(group.ID))
	}
	return types.SetValueMust(types.Int64Type, elems)
}

func securityGroupIDs(set types.Set) []int {
	ids := make([]int, 0, len(set.Elements()))
	for _, elem := range set.Elements() {
		if id, ok := elem.(types.Int64); ok && !id.IsNull() && !id.IsUnknown() {
			ids = append(ids, int(id.ValueInt64()))
		}
	}
	return ids
}

func (c computeNetworkInterfaceResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importStateCompositeInt64IDs(ctx, request, response, path.Root("server_id"), path.Root("id"))
}
