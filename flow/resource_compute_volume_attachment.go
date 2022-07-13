package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/compute"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ tfsdk.ResourceType = (*computeVolumeAttachmentResourceType)(nil)
var _ tfsdk.Resource = (*computeVolumeAttachmentResource)(nil)
var _ tfsdk.ResourceWithImportState = (*computeVolumeAttachmentResource)(nil)

type computeVolumeAttachmentResourceData struct {
	VolumeID types.Int64 `tfsdk:"volume_id"`
	ServerID types.Int64 `tfsdk:"server_id"`
}

func (d *computeVolumeAttachmentResourceData) FromEntity(volume compute.Volume) {
	d.VolumeID = types.Int64{Value: int64(volume.ID)}
	d.ServerID = types.Int64{Value: int64(volume.AttachedTo.ID)}
}

type computeVolumeAttachmentResourceType struct{}

func (t computeVolumeAttachmentResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"volume_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "identifier of the volume for the attachment",
				Required:            true,
			},
			"server_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "identifier of the server for the attachment",
				Required:            true,
			},
		},
	}, nil
}

func (t computeVolumeAttachmentResourceType) NewResource(ctx context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return &computeVolumeAttachmentResource{
		client: prov.client,
	}, nil
}

type computeVolumeAttachmentResource struct {
	client goclient.Client
}

func (r computeVolumeAttachmentResource) Create(ctx context.Context, request tfsdk.CreateResourceRequest, response *tfsdk.CreateResourceResponse) {
	var config computeVolumeAttachmentResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	service := compute.NewVolumeService(r.client)

	volume, err := service.Get(ctx, int(config.VolumeID.Value))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get volume: %s", err))
		return
	}

	// volume is already attached to the requested server -> requested state is already present
	if volume.AttachedTo.ID == int(config.ServerID.Value) {
		var state computeVolumeAttachmentResourceData
		state.FromEntity(volume)

		diagnostics = response.State.Set(ctx, state)
		response.Diagnostics.Append(diagnostics...)
		return
	}

	// volume is already attached to a different server
	if volume.AttachedTo.ID != 0 {
		response.Diagnostics.AddError("Volume Already Attached", "volume is already attached to a different server")
		return
	}

	// volume is not attached to any server yet -> attach it
	attach := compute.VolumeAttach{
		InstanceID: int(config.ServerID.Value),
	}

	volume, err = service.Attach(ctx, int(config.VolumeID.Value), attach)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to attach volume: %s", err))
		return
	}

	var state computeVolumeAttachmentResourceData
	state.FromEntity(volume)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (r computeVolumeAttachmentResource) Read(ctx context.Context, request tfsdk.ReadResourceRequest, response *tfsdk.ReadResourceResponse) {
	var state computeVolumeAttachmentResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	volume, err := compute.NewVolumeService(r.client).Get(ctx, int(state.VolumeID.Value))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get volume: %s", err))
		return
	}

	state.FromEntity(volume)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (r computeVolumeAttachmentResource) Update(ctx context.Context, request tfsdk.UpdateResourceRequest, response *tfsdk.UpdateResourceResponse) {
	var state computeVolumeAttachmentResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	var config computeVolumeAttachmentResourceData
	diagnostics = request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	// detach the volume from the current server
	err := compute.NewVolumeService(r.client).Detach(ctx, int(state.VolumeID.Value), int(state.ServerID.Value))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to detach volume from current server: %s", err))
		return
	}

	tflog.Trace(ctx, "volume attachment: volume detached from previous server")

	// attach the volume to the new server
	attach := compute.VolumeAttach{
		InstanceID: int(config.ServerID.Value),
	}

	volume, err := compute.NewVolumeService(r.client).Attach(ctx, int(state.VolumeID.Value), attach)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to attach volume to new server: %s", err))
		return
	}

	tflog.Trace(ctx, "volume attachment: volume attached to new server")

	state.FromEntity(volume)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (r computeVolumeAttachmentResource) Delete(ctx context.Context, request tfsdk.DeleteResourceRequest, response *tfsdk.DeleteResourceResponse) {
	var state computeVolumeAttachmentResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	err := compute.NewVolumeService(r.client).Detach(ctx, int(state.VolumeID.Value), int(state.ServerID.Value))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to detach volume: %s", err))
		return
	}
}

func (r computeVolumeAttachmentResource) ImportState(ctx context.Context, request tfsdk.ImportResourceStateRequest, response *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, tftypes.NewAttributePath().WithAttributeName("volume_id"), request, response)
}
