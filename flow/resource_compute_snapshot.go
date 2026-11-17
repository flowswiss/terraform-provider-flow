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

var (
	_ resource.Resource                = (*computeSnapshotResource)(nil)
	_ resource.ResourceWithConfigure   = (*computeSnapshotResource)(nil)
	_ resource.ResourceWithImportState = (*computeSnapshotResource)(nil)
)

type computeSnapshotResourceData struct {
	ID        types.Int64  `tfsdk:"id"`
	Size      types.Int64  `tfsdk:"size"`
	CreatedAt types.String `tfsdk:"created_at"`

	Name     types.String `tfsdk:"name"`
	VolumeID types.Int64  `tfsdk:"volume_id"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (d *computeSnapshotResourceData) FromEntity(snapshot compute.Snapshot) {
	d.ID = types.Int64Value(int64(snapshot.ID))
	d.Size = types.Int64Value(int64(snapshot.Size))
	d.CreatedAt = types.StringValue(snapshot.CreatedAt.String())

	d.Name = types.StringValue(snapshot.Name)
	d.VolumeID = types.Int64Value(int64(snapshot.Volume.ID))
}

func (t computeSnapshotResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the snapshot",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"size": schema.Int64Attribute{
				MarkdownDescription: "size of the snapshot in GiB",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "date and time when the snapshot was created",
				Computed:            true,
			},

			"name": schema.StringAttribute{
				MarkdownDescription: "name of the snapshot",
				Required:            true,
			},
			"volume_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the volume",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create:            true,
				CreateDescription: timeoutDescription("bounds the whole create; unset, the snapshot is given 30m to become available, which scales with the volume size"),
				Delete:            true,
				DeleteDescription: timeoutDescription("bounds the whole delete; unset, the snapshot is given 10m to disappear"),
			}),
		},
	}
}

func newComputeSnapshotResource() resource.Resource {
	return &computeSnapshotResource{}
}

func (r *computeSnapshotResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_snapshot"
}

func (r *computeSnapshotResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	r.client = client
}

type computeSnapshotResource struct {
	client flowClient
}

func (r computeSnapshotResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var config computeSnapshotResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	ctx, cancel := withTimeout(ctx, config.Timeouts.Create, &response.Diagnostics)
	defer cancel()

	create := compute.SnapshotCreateReq{
		Name:     config.Name.ValueString(),
		VolumeID: int(config.VolumeID.ValueInt64()),
	}

	var snapshot compute.Snapshot
	err := retryCreate(ctx, "create snapshot", func() (err error) {
		snapshot, err = r.client.Compute.Snapshot.Create(ctx, create)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create snapshot: %s", err))
		return
	}

	tflog.Trace(ctx, "created snapshot", map[string]interface{}{
		"id":   snapshot.ID,
		"data": snapshot,
	})

	available, err := r.waitForSnapshotAvailable(ctx, snapshot.ID)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for snapshot to be available: %s", err))
		// No return: the snapshot already exists. Save it with the error, so
		// Terraform replaces it on the next apply instead of forgetting it.
	}
	if available.ID != 0 {
		snapshot = available
	}

	var state computeSnapshotResourceData
	state.FromEntity(snapshot)
	state.Timeouts = config.Timeouts

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (r computeSnapshotResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state computeSnapshotResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	snapshot, err := r.client.Compute.Snapshot.Get(ctx, compute.SnapshotGetReq{ID: uint(state.ID.ValueInt64())})
	if err != nil {
		if isNotFound(err) {
			removeGone(ctx, response, fmt.Sprintf("snapshot %d", state.ID.ValueInt64()))
			return
		}
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get snapshot: %s", err))
		return
	}

	state.FromEntity(snapshot)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (r computeSnapshotResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
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
		state.Timeouts = config.Timeouts
		response.Diagnostics.Append(response.State.Set(ctx, state)...)
		return
	}

	tflog.Debug(ctx, "snapshot name has changed: updating snapshot", map[string]interface{}{
		"snapshot_id":    state.ID,
		"previous_name":  state.Name,
		"requested_name": config.Name,
	})

	update := compute.SnapshotUpdateReq{
		ID:   uint(state.ID.ValueInt64()),
		Name: nonZero(config.Name.ValueString()),
	}

	var snapshot compute.Snapshot
	err := retry(ctx, "update snapshot", func() (err error) {
		snapshot, err = r.client.Compute.Snapshot.Update(ctx, update)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update snapshot: %s", err))
		return
	}

	state.FromEntity(snapshot)
	state.Timeouts = config.Timeouts

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (r computeSnapshotResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state computeSnapshotResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	ctx, cancel := withTimeout(ctx, state.Timeouts.Delete, &response.Diagnostics)
	defer cancel()

	snapshotID := int(state.ID.ValueInt64())

	err := retryDelete(ctx, "delete snapshot", func() error {
		return r.client.Compute.Snapshot.Delete(ctx, compute.SnapshotDeleteReq{ID: uint(snapshotID)})
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete snapshot: %s", err))
		return
	}

	err = waitForGone(ctx, goneTimeout, fmt.Sprintf("snapshot %d", snapshotID), func(ctx context.Context) error {
		_, err := r.client.Compute.Snapshot.Get(ctx, compute.SnapshotGetReq{ID: uint(snapshotID)})
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for snapshot deletion: %s", err))
		return
	}
}

func (r computeSnapshotResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importStatePassthroughInt64ID(ctx, path.Root("id"), request, response)
}

func (r computeSnapshotResource) waitForSnapshotAvailable(ctx context.Context, snapshotID int) (snapshot compute.Snapshot, err error) {
	err = waitFor(ctx, snapshotTimeout, defaultWaitInterval, fmt.Sprintf("snapshot %d to be available", snapshotID), func(ctx context.Context) (bool, error) {
		got, err := r.client.Compute.Snapshot.Get(ctx, compute.SnapshotGetReq{ID: uint(snapshotID)})
		if err != nil {
			return false, err
		}
		snapshot = got

		switch snapshot.Status.ID {
		case compute.SnapshotStatusAvailable:
			return true, nil
		case compute.SnapshotStatusError:
			return false, fmt.Errorf("snapshot %d is in error state", snapshotID)
		default:
			return false, nil
		}
	})

	return snapshot, err
}
