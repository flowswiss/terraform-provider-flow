package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/compute"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ tfsdk.ResourceType            = (*computeVolumeResourceType)(nil)
	_ tfsdk.Resource                = (*computeVolumeResource)(nil)
	_ tfsdk.ResourceWithImportState = (*computeVolumeResource)(nil)
)

type computeVolumeResourceData struct {
	ID           types.Int64  `tfsdk:"id"`
	SerialNumber types.String `tfsdk:"serial_number"`
	Name         types.String `tfsdk:"name"`
	Size         types.Int64  `tfsdk:"size"`
	Location     types.Int64  `tfsdk:"location_id"`
	Snapshot     types.Int64  `tfsdk:"restore_from_snapshot_id"`
}

func (d *computeVolumeResourceData) FromEntity(volume compute.Volume) {
	d.ID = types.Int64{Value: int64(volume.ID)}
	d.SerialNumber = types.String{Value: volume.SerialNumber}
	d.Name = types.String{Value: volume.Name}
	d.Size = types.Int64{Value: int64(volume.Size)}
	d.Location = types.Int64{Value: int64(volume.Location.ID)}
}

type computeVolumeResourceType struct{}

func (t computeVolumeResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the volume",
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},
			"serial_number": {
				Type:                types.StringType,
				MarkdownDescription: "unique serial number of the volume",
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},

			"name": {
				Type:                types.StringType,
				MarkdownDescription: "name of the volume",
				Optional:            true,
			},
			"size": {
				Type:                types.Int64Type,
				MarkdownDescription: "size in GiB of the volume",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					// TODO not sure whether this should trigger a recreate since the data on the volume will be lost
					tfsdk.RequiresReplaceIf(func(ctx context.Context, state, config attr.Value, path *tftypes.AttributePath) (bool, diag.Diagnostics) {
						return state.(types.Int64).Value > config.(types.Int64).Value, nil
					}, "", "volume size cannot be decreased"),
				},
			},
			"location_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "identifier of the location of the volume",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"restore_from_snapshot_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "restore the volume from the snapshot",
				Optional:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
		},
	}, nil
}

func (t computeVolumeResourceType) NewResource(ctx context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return computeVolumeResource{
		volumeService: compute.NewVolumeService(prov.client),
	}, diagnostics
}

type computeVolumeResource struct {
	volumeService compute.VolumeService
}

func (r computeVolumeResource) Create(ctx context.Context, request tfsdk.CreateResourceRequest, response *tfsdk.CreateResourceResponse) {
	var config computeVolumeResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	create := compute.VolumeCreate{
		Name:       config.Name.Value,
		Size:       int(config.Size.Value),
		LocationID: int(config.Location.Value),
		SnapshotID: int(config.Snapshot.Value),
	}

	volume, err := r.volumeService.Create(ctx, create)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create volume: %s", err))
		return
	}

	var state computeVolumeResourceData
	state.FromEntity(volume)

	// copy the restored snapshot property from the config. in the api we don't know anymore if there was a snapshot
	// that has been restored.
	state.Snapshot = config.Snapshot

	tflog.Trace(ctx, "created volume", map[string]interface{}{
		"id":   volume.ID,
		"data": volume,
	})

	if volume.Status.ID == compute.VolumeStatusWorking {
		// wait for the volume to be ready
		waitForCondition(ctx, func(ctx context.Context) (bool, diag.Diagnostics) {
			return r.waitForVolumeStatus(ctx, volume.ID)
		})
	}

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (r computeVolumeResource) Read(ctx context.Context, request tfsdk.ReadResourceRequest, response *tfsdk.ReadResourceResponse) {
	var state computeVolumeResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	volume, err := r.volumeService.Get(ctx, int(state.ID.Value))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get volume: %s", err))
		return
	}

	state.FromEntity(volume)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (r computeVolumeResource) Update(ctx context.Context, request tfsdk.UpdateResourceRequest, response *tfsdk.UpdateResourceResponse) {
	var state computeVolumeResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	var config computeVolumeResourceData
	diagnostics = request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	volume, err := r.volumeService.Get(ctx, int(state.ID.Value))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get volume: %s", err))
		return
	}

	if !config.Name.Equal(state.Name) {
		tflog.Debug(ctx, "volume name has changed: updating volume", map[string]interface{}{
			"volume_id":      state.ID,
			"previous_name":  state.Name,
			"requested_name": config.Name,
		})

		update := compute.VolumeUpdate{
			Name: config.Name.Value,
		}

		volume, err = r.volumeService.Update(ctx, int(state.ID.Value), update)
		if err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update volume: %s", err))
			return
		}
	}

	if !config.Size.Equal(state.Size) {
		tflog.Debug(ctx, "volume size has changed: expanding volume", map[string]interface{}{
			"volume_id":      state.ID,
			"previous_size":  state.Size,
			"requested_size": config.Size,
		})

		expand := compute.VolumeExpand{
			Size: int(config.Size.Value),
		}

		volume, err = r.volumeService.Expand(ctx, int(state.ID.Value), expand)
		if err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to expand volume: %s", err))
			return
		}
	}

	state.FromEntity(volume)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (r computeVolumeResource) Delete(ctx context.Context, request tfsdk.DeleteResourceRequest, response *tfsdk.DeleteResourceResponse) {
	var state computeVolumeResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	err := r.volumeService.Delete(ctx, int(state.ID.Value))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete volume: %s", err))
		return
	}
}

func (r computeVolumeResource) ImportState(ctx context.Context, request tfsdk.ImportResourceStateRequest, response *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, tftypes.NewAttributePath().WithAttributeName("id"), request, response)
}

func (r computeVolumeResource) waitForVolumeStatus(ctx context.Context, volumeID int) (done bool, diagnostics diag.Diagnostics) {
	volume, err := r.volumeService.Get(ctx, volumeID)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("unable to get volume: %s", err))
		return
	}

	done = volume.Status.ID != compute.VolumeStatusWorking
	return
}
