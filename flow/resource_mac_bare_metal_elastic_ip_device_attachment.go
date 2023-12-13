package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/macbaremetal"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ tfsdk.ResourceType = (*macBareMetalElasticIPDeviceAttachmentResourceType)(nil)
	_ tfsdk.Resource     = (*macBareMetalElasticIPDeviceAttachmentResource)(nil)
)

type macBareMetalElasticIPDeviceAttachmentResourceData struct {
	DeviceID           types.Int64 `tfsdk:"device_id"`
	NetworkInterfaceID types.Int64 `tfsdk:"network_interface_id"`
	ElasticIPID        types.Int64 `tfsdk:"elastic_ip_id"`
}

func (m *macBareMetalElasticIPDeviceAttachmentResourceData) FromEntity(
	device macbaremetal.Device,
	elasticIP macbaremetal.ElasticIP,
) {
	m.DeviceID = types.Int64{Value: int64(device.ID)}
	m.NetworkInterfaceID = types.Int64{Null: true}
	m.ElasticIPID = types.Int64{Value: int64(elasticIP.ID)}

	for _, iface := range device.NetworkInterfaces {
		if iface.PublicIP == elasticIP.PublicIP {
			m.NetworkInterfaceID = types.Int64{Value: int64(iface.ID)}
		}
	}
}

type macBareMetalElasticIPDeviceAttachmentResourceType struct{}

func (m macBareMetalElasticIPDeviceAttachmentResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"device_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the device to attach the elastic ip to",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"network_interface_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the network interface of the device to attach the elastic ip to",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"elastic_ip_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the elastic ip to attach to the device",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
		},
	}, nil
}

func (m macBareMetalElasticIPDeviceAttachmentResourceType) NewResource(ctx context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return macBareMetalElasticIPDeviceAttachmentResource{
		deviceService:    macbaremetal.NewDeviceService(prov.client),
		elasticIPService: macbaremetal.NewElasticIPService(prov.client),
		client:           prov.client,
	}, diagnostics
}

type macBareMetalElasticIPDeviceAttachmentResource struct {
	deviceService    macbaremetal.DeviceService
	elasticIPService macbaremetal.ElasticIPService

	client goclient.Client
}

func (c macBareMetalElasticIPDeviceAttachmentResource) Create(ctx context.Context, request tfsdk.CreateResourceRequest, response *tfsdk.CreateResourceResponse) {
	var config macBareMetalElasticIPDeviceAttachmentResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	serverID := int(config.DeviceID.Value)

	attach := macbaremetal.ElasticIPAttach{
		ElasticIPID:        int(config.ElasticIPID.Value),
		NetworkInterfaceID: int(config.NetworkInterfaceID.Value),
	}

	elasticIP, err := macbaremetal.NewAttachedElasticIPService(c.client, serverID).Attach(ctx, attach)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to attach elastic ip: %s", err))
		return
	}

	device, err := macbaremetal.NewDeviceService(c.client).Get(ctx, serverID)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get device: %s", err))
		return
	}

	var state macBareMetalElasticIPDeviceAttachmentResourceData
	state.FromEntity(device, elasticIP)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c macBareMetalElasticIPDeviceAttachmentResource) Read(ctx context.Context, request tfsdk.ReadResourceRequest, response *tfsdk.ReadResourceResponse) {
	var state macBareMetalElasticIPDeviceAttachmentResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	device, err := macbaremetal.NewDeviceService(c.client).Get(ctx, int(state.DeviceID.Value))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get device: %s", err))
		return
	}

	elasticIP, diagnostics := findMacBareMetalElasticIP(ctx, c.elasticIPService, int(state.ElasticIPID.Value))
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	state.FromEntity(device, elasticIP)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c macBareMetalElasticIPDeviceAttachmentResource) Update(ctx context.Context, request tfsdk.UpdateResourceRequest, response *tfsdk.UpdateResourceResponse) {
	response.Diagnostics.AddError("Not Supported", "updating an elastic ip attachment is not supported")
}

func (c macBareMetalElasticIPDeviceAttachmentResource) Delete(ctx context.Context, request tfsdk.DeleteResourceRequest, response *tfsdk.DeleteResourceResponse) {
	var state macBareMetalElasticIPDeviceAttachmentResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	err := macbaremetal.NewAttachedElasticIPService(c.client, int(state.DeviceID.Value)).Detach(ctx, int(state.ElasticIPID.Value))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to detach elastic ip: %s", err))
		return
	}
}
