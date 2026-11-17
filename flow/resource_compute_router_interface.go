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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*computeRouterInterfaceResource)(nil)
	_ resource.ResourceWithConfigure   = (*computeRouterInterfaceResource)(nil)
	_ resource.ResourceWithImportState = (*computeRouterInterfaceResource)(nil)
)

type computeRouterInterfaceResourceData struct {
	ID        types.Int64  `tfsdk:"id"`
	RouterID  types.Int64  `tfsdk:"router_id"`
	NetworkID types.Int64  `tfsdk:"network_id"`
	PrivateIP types.String `tfsdk:"private_ip"`
}

func (c *computeRouterInterfaceResourceData) FromEntity(routerID int, routerInterface compute.RouterInterface) {
	c.ID = types.Int64Value(int64(routerInterface.ID))
	c.RouterID = types.Int64Value(int64(routerID))
	c.PrivateIP = types.StringValue(routerInterface.PrivateIP)
	c.NetworkID = types.Int64Value(int64(routerInterface.Network.ID))
}

func (c computeRouterInterfaceResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Import: `terraform import flow_compute_router_interface.<name> <router_id>:<id>`",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the router interface",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"router_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the router",
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
				MarkdownDescription: "private IP address of the router interface",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func newComputeRouterInterfaceResource() resource.Resource {
	return &computeRouterInterfaceResource{}
}

func (c *computeRouterInterfaceResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_router_interface"
}

func (c *computeRouterInterfaceResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.client = client
}

type computeRouterInterfaceResource struct {
	client flowClient
}

func (c computeRouterInterfaceResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var config computeRouterInterfaceResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	routerID := int(config.RouterID.ValueInt64())
	create := compute.RouterInterfaceCreateReq{
		RouterID:  uint(routerID),
		NetworkID: int(config.NetworkID.ValueInt64()),
		PrivateIP: config.PrivateIP.ValueString(),
	}

	var routerInterface compute.RouterInterface
	err := retryCreate(ctx, "create router interface", func() (err error) {
		routerInterface, err = c.client.Compute.RouterInterface.Create(ctx, create)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create router interface: %s", err))
		return
	}

	var state computeRouterInterfaceResourceData
	state.FromEntity(routerID, routerInterface)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeRouterInterfaceResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state computeRouterInterfaceResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	routerID := int(state.RouterID.ValueInt64())
	list, err := c.client.Compute.RouterInterface.List(ctx, compute.RouterInterfaceListReq{
		RouterID: uint(routerID),
		Cursor:   core.CursorAll,
	})
	if err != nil {
		if isNotFound(err) {
			removeGone(ctx, response, fmt.Sprintf("router %d", routerID))
			return
		}
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list router interfaces: %s", err))
		return
	}

	for _, routerInterface := range list.Items {
		if routerInterface.ID == int(state.ID.ValueInt64()) {
			state.FromEntity(routerID, routerInterface)

			diagnostics = response.State.Set(ctx, state)
			response.Diagnostics.Append(diagnostics...)
			return
		}
	}

	removeGone(ctx, response, fmt.Sprintf("router interface %d", state.ID.ValueInt64()))
}

func (c computeRouterInterfaceResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	response.Diagnostics.AddError("Not Supported", "updating a router interface is not supported")
}

func (c computeRouterInterfaceResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state computeRouterInterfaceResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	err := retryDelete(ctx, "delete router interface", func() error {
		return c.client.Compute.RouterInterface.Delete(ctx, compute.RouterInterfaceDeleteReq{
			RouterID:          uint(state.RouterID.ValueInt64()),
			RouterInterfaceID: uint(state.ID.ValueInt64()),
		})
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete router interface: %s", err))
		return
	}
}

func (c computeRouterInterfaceResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importStateCompositeInt64IDs(ctx, request, response, path.Root("router_id"), path.Root("id"))
}
