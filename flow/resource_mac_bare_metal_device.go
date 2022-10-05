package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/common"
	"github.com/flowswiss/goclient/macbaremetal"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ tfsdk.ResourceType            = (*macBareMetalDeviceResourceType)(nil)
	_ tfsdk.Resource                = (*macBareMetalDeviceResource)(nil)
	_ tfsdk.ResourceWithImportState = (*macBareMetalDeviceResource)(nil)
)

type macBareMetalDeviceResourceData struct {
	ID         types.Int64  `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	LocationID types.Int64  `tfsdk:"location_id"`
	ProductID  types.Int64  `tfsdk:"product_id"`
	NetworkID  types.Int64  `tfsdk:"network_id"`
	Password   types.String `tfsdk:"password"`
}

func (m *macBareMetalDeviceResourceData) FromEntity(device macbaremetal.Device) {
	m.ID = types.Int64{Value: int64(device.ID)}
	m.Name = types.String{Value: device.Name}
	m.LocationID = types.Int64{Value: int64(device.Location.ID)}
	m.ProductID = types.Int64{Value: int64(device.Product.ID)}
	m.NetworkID = types.Int64{Value: int64(device.Network.ID)}
}

type macBareMetalDeviceResourceType struct{}

func (m macBareMetalDeviceResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the device",
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},
			"name": {
				Type:                types.StringType,
				MarkdownDescription: "name of the device",
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
			"network_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the network",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
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
			"password": {
				Type:                types.StringType,
				MarkdownDescription: "password of the device",
				Required:            true,
				Sensitive:           true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
		},
	}, nil
}

func (m macBareMetalDeviceResourceType) NewResource(ctx context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return macBareMetalDeviceResource{
		orderService:  common.NewOrderService(prov.client),
		deviceService: macbaremetal.NewDeviceService(prov.client),
	}, diagnostics
}

type macBareMetalDeviceResource struct {
	orderService  common.OrderService
	deviceService macbaremetal.DeviceService
}

func (m macBareMetalDeviceResource) Create(ctx context.Context, request tfsdk.CreateResourceRequest, response *tfsdk.CreateResourceResponse) {
	var config macBareMetalDeviceResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	create := macbaremetal.DeviceCreate{
		Name:            config.Name.Value,
		LocationID:      int(config.LocationID.Value),
		ProductID:       int(config.ProductID.Value),
		NetworkID:       int(config.NetworkID.Value),
		AttachElasticIP: false,
		Password:        config.Password.Value,
	}

	ordering, err := m.deviceService.Create(ctx, create)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create device: %s", err))
		return
	}

	order, err := m.orderService.WaitUntilProcessed(ctx, ordering)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for device creation: %s", err))
		return
	}

	device, err := m.deviceService.Get(ctx, order.Product.ID)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get device: %s", err))
		return
	}

	var state macBareMetalDeviceResourceData
	state.FromEntity(device)

	state.Password = config.Password

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (m macBareMetalDeviceResource) Read(ctx context.Context, request tfsdk.ReadResourceRequest, response *tfsdk.ReadResourceResponse) {
	var state macBareMetalDeviceResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	device, err := m.deviceService.Get(ctx, int(state.ID.Value))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get device: %s", err))
		return
	}

	state.FromEntity(device)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (m macBareMetalDeviceResource) Update(ctx context.Context, request tfsdk.UpdateResourceRequest, response *tfsdk.UpdateResourceResponse) {
	var state macBareMetalDeviceResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	var config macBareMetalDeviceResourceData
	diagnostics = request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	update := macbaremetal.DeviceUpdate{
		Name: config.Name.Value,
	}

	device, err := m.deviceService.Update(ctx, int(state.ID.Value), update)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update device: %s", err))
		return
	}

	state.FromEntity(device)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (m macBareMetalDeviceResource) Delete(ctx context.Context, request tfsdk.DeleteResourceRequest, response *tfsdk.DeleteResourceResponse) {
	var state macBareMetalDeviceResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	err := m.deviceService.Delete(ctx, int(state.ID.Value))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete device: %s", err))
		return
	}
}

func (m macBareMetalDeviceResource) ImportState(ctx context.Context, request tfsdk.ImportResourceStateRequest, response *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, path.Root("id"), request, response)
}
