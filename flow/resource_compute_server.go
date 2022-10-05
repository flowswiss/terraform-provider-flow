package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/common"
	"github.com/flowswiss/goclient/compute"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ tfsdk.ResourceType            = (*computeServerResourceType)(nil)
	_ tfsdk.Resource                = (*computeServerResource)(nil)
	_ tfsdk.ResourceWithImportState = (*computeServerResource)(nil)
)

type computeServerResourceData struct {
	ID         types.Int64  `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	LocationID types.Int64  `tfsdk:"location_id"`
	ImageID    types.Int64  `tfsdk:"image_id"`
	ProductID  types.Int64  `tfsdk:"product_id"`
	NetworkID  types.Int64  `tfsdk:"network_id"`
	PrivateIP  types.String `tfsdk:"private_ip"`
	KeyPairID  types.Int64  `tfsdk:"key_pair_id"`
	Password   types.String `tfsdk:"password"`
	CloudInit  types.String `tfsdk:"cloud_init"`
}

func (c *computeServerResourceData) FromEntity(server compute.Server) {
	c.ID = types.Int64{Value: int64(server.ID)}
	c.Name = types.String{Value: server.Name}
	c.LocationID = types.Int64{Value: int64(server.Location.ID)}
	c.ImageID = types.Int64{Value: int64(server.Image.ID)}
	c.ProductID = types.Int64{Value: int64(server.Product.ID)}
	c.KeyPairID = types.Int64{Value: int64(server.KeyPair.ID)}

	if len(server.Networks) != 0 {
		network := server.Networks[0]
		c.NetworkID = types.Int64{Value: int64(network.ID)}
		c.PrivateIP = types.String{Value: network.Interfaces[0].PrivateIP}
	}
}

type computeServerResourceType struct{}

func (c computeServerResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the server",
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},
			"name": {
				Type:                types.StringType,
				MarkdownDescription: "name of the server",
				Required:            true,
			},
			"location_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the location",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"image_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the image",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"product_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the product",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"network_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the initial network",
				Optional:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"private_ip": {
				Type:                types.StringType,
				MarkdownDescription: "initial private ip of the server",
				Optional:            true,
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"key_pair_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the key pair",
				Optional:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"password": {
				Type:                types.StringType,
				MarkdownDescription: "initial windows password of the server",
				Optional:            true,
				Sensitive:           true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"cloud_init": {
				Type:                types.StringType,
				MarkdownDescription: "cloud init script",
				Optional:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
		},
	}, nil
}

func (c computeServerResourceType) NewResource(ctx context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return computeServerResource{
		serverService: compute.NewServerService(prov.client),
		orderService:  common.NewOrderService(prov.client),
	}, diagnostics
}

type computeServerResource struct {
	serverService compute.ServerService
	orderService  common.OrderService
}

func (c computeServerResource) Create(ctx context.Context, request tfsdk.CreateResourceRequest, response *tfsdk.CreateResourceResponse) {
	var config computeServerResourceData
	response.Diagnostics.Append(request.Config.Get(ctx, &config)...)
	if response.Diagnostics.HasError() {
		return
	}

	create := compute.ServerCreate{
		Name:             config.Name.Value,
		LocationID:       int(config.LocationID.Value),
		ImageID:          int(config.ImageID.Value),
		ProductID:        int(config.ProductID.Value),
		AttachExternalIP: false,
		NetworkID:        int(config.NetworkID.Value),
		PrivateIP:        config.PrivateIP.Value,
		KeyPairID:        int(config.KeyPairID.Value),
		Password:         config.Password.Value,
		CloudInit:        config.CloudInit.Value,
	}

	ordering, err := c.serverService.Create(ctx, create)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create server: %s", err))
		return
	}

	order, err := c.orderService.WaitUntilProcessed(ctx, ordering)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for server creation: %s", err))
		return
	}

	server, err := c.serverService.Get(ctx, order.Product.ID)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get server: %s", err))
		return
	}

	var state computeServerResourceData
	state.FromEntity(server)

	state.Password = config.Password
	state.CloudInit = config.CloudInit

	response.Diagnostics.Append(response.State.Set(ctx, state)...)
}

func (c computeServerResource) Read(ctx context.Context, request tfsdk.ReadResourceRequest, response *tfsdk.ReadResourceResponse) {
	var state computeServerResourceData
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	server, err := c.serverService.Get(ctx, int(state.ID.Value))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get server: %s", err))
		return
	}

	state.FromEntity(server)

	response.Diagnostics.Append(response.State.Set(ctx, state)...)
}

func (c computeServerResource) Update(ctx context.Context, request tfsdk.UpdateResourceRequest, response *tfsdk.UpdateResourceResponse) {
	var state computeServerResourceData
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	var config computeServerResourceData
	response.Diagnostics.Append(request.Config.Get(ctx, &config)...)
	if response.Diagnostics.HasError() {
		return
	}

	update := compute.ServerUpdate{
		Name: config.Name.Value,
	}

	server, err := c.serverService.Update(ctx, int(state.ID.Value), update)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update server: %s", err))
		return
	}

	state.FromEntity(server)

	response.Diagnostics.Append(response.State.Set(ctx, state)...)
}

func (c computeServerResource) Delete(ctx context.Context, request tfsdk.DeleteResourceRequest, response *tfsdk.DeleteResourceResponse) {
	var state computeServerResourceData
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	err := c.serverService.Delete(ctx, int(state.ID.Value), false)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete server: %s", err))
		return
	}
}

func (c computeServerResource) ImportState(ctx context.Context, request tfsdk.ImportResourceStateRequest, response *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, path.Root("id"), request, response)
}
