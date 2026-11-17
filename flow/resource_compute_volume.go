package flow

import (
	"context"
	"fmt"
	"time"

	"github.com/flowswiss/goclient/v2/compute"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = (*computeVolumeResource)(nil)
	_ resource.ResourceWithConfigure   = (*computeVolumeResource)(nil)
	_ resource.ResourceWithImportState = (*computeVolumeResource)(nil)
)

type computeVolumeResourceData struct {
	ID           types.Int64  `tfsdk:"id"`
	SerialNumber types.String `tfsdk:"serial_number"`
	Name         types.String `tfsdk:"name"`
	Size         types.Int64  `tfsdk:"size"`
	Location     types.Int64  `tfsdk:"location_id"`
	Snapshot     types.Int64  `tfsdk:"restore_from_snapshot_id"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (d *computeVolumeResourceData) FromEntity(volume compute.Volume) {
	d.ID = types.Int64Value(int64(volume.ID))
	d.SerialNumber = types.StringValue(volume.SerialNumber)
	d.Name = types.StringValue(volume.Name)
	d.Size = types.Int64Value(int64(volume.Size))
	d.Location = types.Int64Value(int64(volume.Location.ID))
}

func (t computeVolumeResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the volume",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"serial_number": schema.StringAttribute{
				MarkdownDescription: "unique serial number of the volume",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"name": schema.StringAttribute{
				MarkdownDescription: "name of the volume",
				Required:            true,
			},
			"size": schema.Int64Attribute{
				MarkdownDescription: "size in GiB of the volume",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplaceIf(func(ctx context.Context, request planmodifier.Int64Request, response *int64planmodifier.RequiresReplaceIfFuncResponse) {
						response.RequiresReplace = request.StateValue.ValueInt64() > request.PlanValue.ValueInt64()
					}, "", "volume size cannot be decreased"),
				},
			},
			"location_id": schema.Int64Attribute{
				MarkdownDescription: "identifier of the location of the volume",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"restore_from_snapshot_id": schema.Int64Attribute{
				MarkdownDescription: "restore the volume from the snapshot",
				Optional:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create:            true,
				CreateDescription: timeoutDescription("bounds the whole create; unset, the volume is given 30m to settle, which a restore from a snapshot needs"),
				Update:            true,
				UpdateDescription: timeoutDescription("bounds the whole update; unset, an expand gives the volume 5m to settle"),
				Delete:            true,
				DeleteDescription: timeoutDescription("bounds the whole delete; unset, the volume is given 10m to disappear"),
			}),
		},
	}
}

func newComputeVolumeResource() resource.Resource {
	return &computeVolumeResource{}
}

func (r *computeVolumeResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_volume"
}

func (r *computeVolumeResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	r.client = client
}

type computeVolumeResource struct {
	client flowClient
}

func (r computeVolumeResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var config computeVolumeResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	ctx, cancel := withTimeout(ctx, config.Timeouts.Create, &response.Diagnostics)
	defer cancel()

	create := compute.VolumeCreateReq{
		Name:       config.Name.ValueString(),
		Size:       int(config.Size.ValueInt64()),
		LocationID: int(config.Location.ValueInt64()),
		SnapshotID: nonZero(int(config.Snapshot.ValueInt64())),
	}

	var volume compute.Volume
	err := retryCreate(ctx, "create volume", func() (err error) {
		volume, err = r.client.Compute.Volume.Create(ctx, create)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create volume: %s", err))
		return
	}

	tflog.Trace(ctx, "created volume", map[string]interface{}{
		"id":   volume.ID,
		"data": volume,
	})

	settled, err := r.waitForVolumeSettled(ctx, volume.ID, snapshotTimeout)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for volume to settle: %s", err))
	}
	if settled.ID != 0 {
		volume = settled
	}

	var state computeVolumeResourceData
	state.FromEntity(volume)

	state.Snapshot = config.Snapshot
	state.Timeouts = config.Timeouts

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (r computeVolumeResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state computeVolumeResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	volume, err := r.client.Compute.Volume.Get(ctx, compute.VolumeGetReq{ID: uint(state.ID.ValueInt64())})
	if err != nil {
		if isNotFound(err) {
			removeGone(ctx, response, fmt.Sprintf("volume %d", state.ID.ValueInt64()))
			return
		}
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get volume: %s", err))
		return
	}

	state.FromEntity(volume)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (r computeVolumeResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var state computeVolumeResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	var plan computeVolumeResourceData
	diagnostics = request.Plan.Get(ctx, &plan)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	ctx, cancel := withTimeout(ctx, plan.Timeouts.Update, &response.Diagnostics)
	defer cancel()

	volume, err := r.client.Compute.Volume.Get(ctx, compute.VolumeGetReq{ID: uint(state.ID.ValueInt64())})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get volume: %s", err))
		return
	}

	if !plan.Name.Equal(state.Name) {
		tflog.Debug(ctx, "volume name has changed: updating volume", map[string]interface{}{
			"volume_id":      state.ID,
			"previous_name":  state.Name,
			"requested_name": plan.Name,
		})

		update := compute.VolumeUpdateReq{
			ID:   uint(state.ID.ValueInt64()),
			Name: new(plan.Name.ValueString()),
		}

		err = retry(ctx, "update volume", func() (err error) {
			volume, err = r.client.Compute.Volume.Update(ctx, update)
			return err
		})
		if err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update volume: %s", err))
			return
		}
	}

	if !plan.Size.Equal(state.Size) {
		tflog.Debug(ctx, "volume size has changed: expanding volume", map[string]interface{}{
			"volume_id":      state.ID,
			"previous_size":  state.Size,
			"requested_size": plan.Size,
		})

		expand := compute.VolumeExpandReq{
			VolumeID: uint(state.ID.ValueInt64()),
			Size:     int(plan.Size.ValueInt64()),
		}

		err = retry(ctx, "expand volume", func() (err error) {
			volume, err = r.client.Compute.Volume.Expand(ctx, expand)
			return err
		})
		if err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to expand volume: %s", err))
			return
		}

		settled, waitErr := r.waitForVolumeSettled(ctx, int(state.ID.ValueInt64()), volumeSettleTimeout)
		if settled.ID != 0 {
			volume = settled
		}
		if waitErr != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for volume to settle: %s", waitErr))
		}
	}

	state.FromEntity(volume)
	state.Timeouts = plan.Timeouts

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (r computeVolumeResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state computeVolumeResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	ctx, cancel := withTimeout(ctx, state.Timeouts.Delete, &response.Diagnostics)
	defer cancel()

	volumeID := int(state.ID.ValueInt64())

	err := retryDelete(ctx, "delete volume", func() error {
		return r.client.Compute.Volume.Delete(ctx, compute.VolumeDeleteReq{ID: uint(volumeID)})
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete volume: %s", err))
		return
	}

	err = waitForGone(ctx, goneTimeout, fmt.Sprintf("volume %d", volumeID), func(ctx context.Context) error {
		_, err := r.client.Compute.Volume.Get(ctx, compute.VolumeGetReq{ID: uint(volumeID)})
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for volume deletion: %s", err))
		return
	}
}

func (r computeVolumeResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importStatePassthroughInt64ID(ctx, path.Root("id"), request, response)
}

func (r computeVolumeResource) waitForVolumeSettled(ctx context.Context, volumeID int, timeout time.Duration) (volume compute.Volume, err error) {
	err = waitFor(ctx, timeout, defaultWaitInterval, fmt.Sprintf("volume %d to settle", volumeID), func(ctx context.Context) (bool, error) {
		got, err := r.client.Compute.Volume.Get(ctx, compute.VolumeGetReq{ID: uint(volumeID)})
		if err != nil {
			return false, err
		}
		volume = got

		switch volume.Status.ID {
		case compute.VolumeStatusAvailable, compute.VolumeStatusInUse:
			return true, nil
		case compute.VolumeStatusError:
			return false, fmt.Errorf("volume %d is in error state", volumeID)
		default:
			return false, nil
		}
	})

	return volume, err
}
