package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/compute"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ tfsdk.ResourceType            = (*computeSnapshotResourceType)(nil)
	_ tfsdk.Resource                = (*computeSnapshotResource)(nil)
	_ tfsdk.ResourceWithImportState = (*computeSnapshotResource)(nil)
)

type computeSnapshotResourceData struct {
	ID        types.Int64  `tfsdk:"id"`
	Size      types.Int64  `tfsdk:"size"`
	CreatedAt types.String `tfsdk:"created_at"`

	Name     types.String `tfsdk:"name"`
	VolumeID types.Int64  `tfsdk:"volume_id"`
}

func (d *computeSnapshotResourceData) FromEntity(snapshot compute.Snapshot) {
	d.ID = types.Int64{Value: int64(snapshot.ID)}
	d.Size = types.Int64{Value: int64(snapshot.Size)}
	d.CreatedAt = types.String{Value: snapshot.CreatedAt.String()}

	d.Name = types.String{Value: snapshot.Name}
	d.VolumeID = types.Int64{Value: int64(snapshot.Volume.ID)}
}

type computeSnapshotResourceType struct{}

func (t computeSnapshotResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the snapshot",
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},
			"size": {
				Type:                types.Int64Type,
				MarkdownDescription: "size of the snapshot in GiB",
				Computed:            true,
			},
			"created_at": {
				Type:                types.StringType,
				MarkdownDescription: "date and time when the snapshot was created",
				Computed:            true,
			},

			"name": {
				Type:                types.StringType,
				MarkdownDescription: "name of the snapshot",
				Required:            true,
			},
			"volume_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the volume",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
		},
	}, nil
}

func (t computeSnapshotResourceType) NewResource(ctx context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return computeSnapshotResource{
		snapshotService: compute.NewSnapshotService(prov.client),
	}, diagnostics
}

type computeSnapshotResource struct {
	snapshotService compute.SnapshotService
}

func (r computeSnapshotResource) Create(ctx context.Context, request tfsdk.CreateResourceRequest, response *tfsdk.CreateResourceResponse) {
	var config computeSnapshotResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	create := compute.SnapshotCreate{
		Name:     config.Name.Value,
		VolumeID: int(config.VolumeID.Value),
	}

	snapshot, err := r.snapshotService.Create(ctx, create)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create snapshot: %s", err))
		return
	}

	var state computeSnapshotResourceData
	state.FromEntity(snapshot)

	tflog.Trace(ctx, "created snapshot", map[string]interface{}{
		"id":   snapshot.ID,
		"data": snapshot,
	})

	if snapshot.Status.ID == compute.SnapshotStatusCreating {
		// wait for the snapshot to be ready
		waitForCondition(ctx, func(ctx context.Context) (bool, diag.Diagnostics) {
			return r.waitForSnapshotStatus(ctx, snapshot.ID)
		})
	}

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (r computeSnapshotResource) Read(ctx context.Context, request tfsdk.ReadResourceRequest, response *tfsdk.ReadResourceResponse) {
	var state computeSnapshotResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	snapshot, err := r.snapshotService.Get(ctx, int(state.ID.Value))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get snapshot: %s", err))
		return
	}

	state.FromEntity(snapshot)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (r computeSnapshotResource) Update(ctx context.Context, request tfsdk.UpdateResourceRequest, response *tfsdk.UpdateResourceResponse) {
	var state computeSnapshotResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	var config computeSnapshotResourceData
	diagnostics = request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	if config.Name.Equal(state.Name) {
		return
	}

	tflog.Debug(ctx, "snapshot name has changed: updating snapshot", map[string]interface{}{
		"snapshot_id":    state.ID,
		"previous_name":  state.Name,
		"requested_name": config.Name,
	})

	update := compute.SnapshotUpdate{
		Name: config.Name.Value,
	}

	snapshot, err := r.snapshotService.Update(ctx, int(state.ID.Value), update)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update snapshot: %s", err))
		return
	}

	state.FromEntity(snapshot)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (r computeSnapshotResource) Delete(ctx context.Context, request tfsdk.DeleteResourceRequest, response *tfsdk.DeleteResourceResponse) {
	var state computeSnapshotResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	err := r.snapshotService.Delete(ctx, int(state.ID.Value))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete snapshot: %s", err))
		return
	}
}

func (r computeSnapshotResource) ImportState(ctx context.Context, request tfsdk.ImportResourceStateRequest, response *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, tftypes.NewAttributePath().WithAttributeName("id"), request, response)
}

func (r computeSnapshotResource) waitForSnapshotStatus(ctx context.Context, snapshotID int) (done bool, diagnostics diag.Diagnostics) {
	snapshot, err := r.snapshotService.Get(ctx, snapshotID)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("unable to get snapshot: %s", err))
		return
	}

	done = snapshot.Status.ID != compute.SnapshotStatusCreating
	return
}
