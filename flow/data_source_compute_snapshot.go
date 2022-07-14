package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/compute"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/flowswiss/terraform-provider-flow/filter"
)

var (
	_ tfsdk.DataSourceType = (*computeSnapshotDataSourceType)(nil)
	_ tfsdk.DataSource     = (*computeSnapshotDataSource)(nil)
)

type computeSnapshotDataSourceData struct {
	ID        types.Int64  `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	VolumeID  types.Int64  `tfsdk:"volume_id"`
	Size      types.Int64  `tfsdk:"size"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func (c *computeSnapshotDataSourceData) FromEntity(snapshot compute.Snapshot) {
	c.ID = types.Int64{Value: int64(snapshot.ID)}
	c.Name = types.String{Value: snapshot.Name}
	c.VolumeID = types.Int64{Value: int64(snapshot.Volume.ID)}
	c.Size = types.Int64{Value: int64(snapshot.Size)}
	c.CreatedAt = types.String{Value: snapshot.CreatedAt.String()}
}

func (c computeSnapshotDataSourceData) AppliesTo(snapshot compute.Snapshot) bool {
	if !c.ID.Null && c.ID.Value != int64(snapshot.ID) {
		return false
	}

	if !c.Name.Null && c.Name.Value != snapshot.Name {
		return false
	}

	if !c.VolumeID.Null && c.VolumeID.Value != int64(snapshot.Volume.ID) {
		return false
	}

	return true
}

type computeSnapshotDataSourceType struct{}

func (c computeSnapshotDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the snapshot",
				Optional:            true,
				Computed:            true,
			},
			"name": {
				Type:                types.StringType,
				MarkdownDescription: "name of the snapshot",
				Optional:            true,
				Computed:            true,
			},
			"volume_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the volume",
				Optional:            true,
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
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
		},
	}, nil
}

func (c computeSnapshotDataSourceType) NewDataSource(ctx context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return computeSnapshotDataSource{
		snapshotService: compute.NewSnapshotService(prov.client),
	}, diagnostics
}

type computeSnapshotDataSource struct {
	snapshotService compute.SnapshotService
}

func (c computeSnapshotDataSource) Read(ctx context.Context, request tfsdk.ReadDataSourceRequest, response *tfsdk.ReadDataSourceResponse) {
	var config computeSnapshotDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := c.snapshotService.List(ctx, goclient.Cursor{NoFilter: 1})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list snapshots: %s", err))
		return
	}

	snapshot, err := filter.FindOne(config, list.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find snapshot: %s", err))
		return
	}

	var state computeSnapshotDataSourceData
	state.FromEntity(snapshot)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
