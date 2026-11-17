package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/v2/compute"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = (*computeVolumeAttachmentResource)(nil)
var _ resource.ResourceWithConfigure = (*computeVolumeAttachmentResource)(nil)
var _ resource.ResourceWithImportState = (*computeVolumeAttachmentResource)(nil)

type computeVolumeAttachmentResourceData struct {
	VolumeID types.Int64 `tfsdk:"volume_id"`
	ServerID types.Int64 `tfsdk:"server_id"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (d *computeVolumeAttachmentResourceData) FromEntity(volume compute.Volume) {
	d.VolumeID = types.Int64Value(int64(volume.ID))
	d.ServerID = types.Int64Value(int64(volume.AttachedTo.ID))
}

func (t computeVolumeAttachmentResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"volume_id": schema.Int64Attribute{
				MarkdownDescription: "identifier of the volume for the attachment — changing it replaces the attachment (detach, attach), the volumes themselves are not touched",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"server_id": schema.Int64Attribute{
				MarkdownDescription: "identifier of the server for the attachment — changing it moves the volume to the other server",
				Required:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create:            true,
				CreateDescription: timeoutDescription("bounds the whole attach; unset, the volume is given 5m to reach in use"),
				Update:            true,
				UpdateDescription: timeoutDescription("bounds detach and re-attach together; unset, each is given 5m"),
				Delete:            true,
				DeleteDescription: timeoutDescription("bounds the whole detach; unset, the volume is given 5m to become available"),
			}),
		},
	}
}

func newComputeVolumeAttachmentResource() resource.Resource {
	return &computeVolumeAttachmentResource{}
}

func (r *computeVolumeAttachmentResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_volume_attachment"
}

func (r *computeVolumeAttachmentResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	r.client = client
}

type computeVolumeAttachmentResource struct {
	client flowClient
}

func (r computeVolumeAttachmentResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var config computeVolumeAttachmentResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	ctx, cancel := withTimeout(ctx, config.Timeouts.Create, &response.Diagnostics)
	defer cancel()

	volume, err := r.client.Compute.Volume.Get(ctx, compute.VolumeGetReq{ID: uint(config.VolumeID.ValueInt64())})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get volume: %s", err))
		return
	}

	if volume.AttachedTo.ID == int(config.ServerID.ValueInt64()) {
		var state computeVolumeAttachmentResourceData
		state.FromEntity(volume)
		state.Timeouts = config.Timeouts

		diagnostics = response.State.Set(ctx, state)
		response.Diagnostics.Append(diagnostics...)
		return
	}

	if volume.AttachedTo.ID != 0 {
		response.Diagnostics.AddError("Volume Already Attached", "volume is already attached to a different server")
		return
	}

	attach := compute.VolumeAttachReq{
		VolumeID:   uint(config.VolumeID.ValueInt64()),
		InstanceID: int(config.ServerID.ValueInt64()),
	}

	err = retry(ctx, "attach volume", func() (err error) {
		volume, err = r.client.Compute.Volume.Attach(ctx, attach)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to attach volume: %s", err))
		return
	}

	attached, err := r.waitForVolumeStatus(ctx, "in use", int(config.VolumeID.ValueInt64()), compute.VolumeStatusInUse)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for volume attachment: %s", err))
		// No return: the attachment already exists. Save it with the error, so
		// Terraform replaces it on the next apply instead of forgetting it.
	}
	if attached.ID != 0 {
		volume = attached
	}

	var state computeVolumeAttachmentResourceData
	state.FromEntity(volume)
	state.Timeouts = config.Timeouts

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (r computeVolumeAttachmentResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state computeVolumeAttachmentResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	volume, err := r.client.Compute.Volume.Get(ctx, compute.VolumeGetReq{ID: uint(state.VolumeID.ValueInt64())})
	if err != nil {
		if isNotFound(err) {
			removeGone(ctx, response, fmt.Sprintf("volume %d", state.VolumeID.ValueInt64()))
			return
		}
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get volume: %s", err))
		return
	}

	if volume.AttachedTo.ID == 0 {
		removeGone(ctx, response, fmt.Sprintf("attachment of volume %d", state.VolumeID.ValueInt64()))
		return
	}

	state.FromEntity(volume)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (r computeVolumeAttachmentResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
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

	if plan.ServerID.Equal(state.ServerID) {
		state.Timeouts = plan.Timeouts

		diagnostics = response.State.Set(ctx, state)
		response.Diagnostics.Append(diagnostics...)
		return
	}

	ctx, cancel := withTimeout(ctx, plan.Timeouts.Update, &response.Diagnostics)
	defer cancel()

	err := retryDelete(ctx, "detach volume", func() error {
		return r.client.Compute.Volume.Detach(ctx, compute.VolumeDetachReq{
			VolumeID:   uint(state.VolumeID.ValueInt64()),
			InstanceID: uint(state.ServerID.ValueInt64()),
		})
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to detach volume from current server: %s", err))
		return
	}

	if _, err := r.waitForVolumeStatus(ctx, "available", int(state.VolumeID.ValueInt64()), compute.VolumeStatusAvailable); err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for volume detachment: %s", err))
		return
	}

	tflog.Trace(ctx, "volume attachment: volume detached from previous server")

	attach := compute.VolumeAttachReq{
		VolumeID:   uint(state.VolumeID.ValueInt64()),
		InstanceID: int(plan.ServerID.ValueInt64()),
	}

	var volume compute.Volume
	err = retry(ctx, "attach volume", func() (err error) {
		volume, err = r.client.Compute.Volume.Attach(ctx, attach)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to attach volume to new server: %s", err))
		return
	}

	attached, waitErr := r.waitForVolumeStatus(ctx, "in use", int(state.VolumeID.ValueInt64()), compute.VolumeStatusInUse)
	if attached.ID != 0 {
		volume = attached
	}
	if waitErr != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for volume attachment: %s", waitErr))
	}

	tflog.Trace(ctx, "volume attachment: volume attached to new server")

	state.FromEntity(volume)
	state.Timeouts = plan.Timeouts

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (r computeVolumeAttachmentResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state computeVolumeAttachmentResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	ctx, cancel := withTimeout(ctx, state.Timeouts.Delete, &response.Diagnostics)
	defer cancel()

	err := retryDelete(ctx, "detach volume", func() error {
		return r.client.Compute.Volume.Detach(ctx, compute.VolumeDetachReq{
			VolumeID:   uint(state.VolumeID.ValueInt64()),
			InstanceID: uint(state.ServerID.ValueInt64()),
		})
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to detach volume: %s", err))
		return
	}

	if _, err := r.waitForVolumeStatus(ctx, "available", int(state.VolumeID.ValueInt64()), compute.VolumeStatusAvailable); err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for volume detachment: %s", err))
		return
	}
}

func (r computeVolumeAttachmentResource) waitForVolumeStatus(ctx context.Context, status string, volumeID, wantStatus int) (volume compute.Volume, err error) {
	err = waitFor(ctx, volumeSettleTimeout, defaultWaitInterval, fmt.Sprintf("volume %d to be %s", volumeID, status), func(ctx context.Context) (bool, error) {
		got, err := r.client.Compute.Volume.Get(ctx, compute.VolumeGetReq{ID: uint(volumeID)})
		if err != nil {
			if wantStatus == compute.VolumeStatusAvailable && isNotFound(err) {
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

func (r computeVolumeAttachmentResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importStatePassthroughInt64ID(ctx, path.Root("volume_id"), request, response)
}
