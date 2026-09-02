package flow

import (
	"context"
	"fmt"
	"net/http"

	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/compute"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
				MarkdownDescription: "identifier of the volume for the attachment — changing it replaces the attachment (detach, attach), the volumes themselves are not touched",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"server_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "identifier of the server for the attachment — changing it moves the volume to the other server",
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

	err = retry(ctx, "attach volume", func() (err error) {
		volume, err = service.Attach(ctx, int(config.VolumeID.Value), attach)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to attach volume: %s", err))
		return
	}

	attached, err := r.waitForVolumeStatus(ctx, "in use", int(config.VolumeID.Value), compute.VolumeStatusInUse)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for volume attachment: %s", err))
	}
	if attached.ID != 0 {
		volume = attached
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
		if isNotFound(err) {
			removeGone(ctx, response, fmt.Sprintf("volume %d", state.VolumeID.Value))
			return
		}
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get volume: %s", err))
		return
	}

	if volume.AttachedTo.ID == 0 {
		removeGone(ctx, response, fmt.Sprintf("attachment of volume %d", state.VolumeID.Value))
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

	var plan computeVolumeAttachmentResourceData
	diagnostics = request.Plan.Get(ctx, &plan)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	// detach the volume from the current server
	err := retryDelete(ctx, "detach volume", func() error {
		return compute.NewVolumeService(r.client).Detach(ctx, int(state.VolumeID.Value), int(state.ServerID.Value))
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to detach volume from current server: %s", err))
		return
	}

	if _, err := r.waitForVolumeStatus(ctx, "available", int(state.VolumeID.Value), compute.VolumeStatusAvailable); err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for volume detachment: %s", err))
		return
	}

	tflog.Trace(ctx, "volume attachment: volume detached from previous server")

	// attach the volume to the new server
	attach := compute.VolumeAttach{
		InstanceID: int(plan.ServerID.Value),
	}

	var volume compute.Volume
	err = retry(ctx, "attach volume", func() (err error) {
		volume, err = compute.NewVolumeService(r.client).Attach(ctx, int(state.VolumeID.Value), attach)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to attach volume to new server: %s", err))
		return
	}

	volume, err = r.waitForVolumeStatus(ctx, "in use", int(state.VolumeID.Value), compute.VolumeStatusInUse)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for volume attachment: %s", err))
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

	err := retryDelete(ctx, "detach volume", func() error {
		return compute.NewVolumeService(r.client).Detach(ctx, int(state.VolumeID.Value), int(state.ServerID.Value))
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to detach volume: %s", err))
		return
	}

	if _, err := r.waitForVolumeStatus(ctx, "available", int(state.VolumeID.Value), compute.VolumeStatusAvailable); err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for volume detachment: %s", err))
		return
	}
}

// the backend waits only on the detach side, and only for 30 seconds; until the
// volume settles, follow-up attach/expand/delete calls are refused
func (r computeVolumeAttachmentResource) waitForVolumeStatus(ctx context.Context, status string, volumeID, wantStatus int) (volume compute.Volume, err error) {
	err = waitFor(ctx, volumeSettleTimeout, defaultWaitInterval, fmt.Sprintf("volume %d to be %s", volumeID, status), func(ctx context.Context) (bool, error) {
		got, err := compute.NewVolumeService(r.client).Get(ctx, volumeID)
		if err != nil {
			// a volume that is gone counts as detached
			if wantStatus == compute.VolumeStatusAvailable && statusCode(err) == http.StatusNotFound {
				return true, nil
			}
			return false, err
		}
		volume = got

		switch volume.Status.ID {
		case wantStatus:
			return true, nil
		case compute.VolumeStatusError:
			return false, fmt.Errorf("volume %d is in error state", volumeID)
		default:
			return false, nil
		}
	})

	return volume, err
}

func (r computeVolumeAttachmentResource) ImportState(ctx context.Context, request tfsdk.ImportResourceStateRequest, response *tfsdk.ImportResourceStateResponse) {
	importStatePassthroughInt64ID(ctx, path.Root("volume_id"), request, response)
}
