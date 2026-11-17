package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/v2/compute"
	"github.com/flowswiss/goclient/v2/core"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/flowswiss/terraform-provider-flow/filter"
)

var (
	_ datasource.DataSource              = (*computeSnapshotDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*computeSnapshotDataSource)(nil)
)

type computeSnapshotDataSourceData struct {
	ID        types.Int64  `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	VolumeID  types.Int64  `tfsdk:"volume_id"`
	Size      types.Int64  `tfsdk:"size"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func (c *computeSnapshotDataSourceData) FromEntity(snapshot compute.Snapshot) {
	c.ID = types.Int64Value(int64(snapshot.ID))
	c.Name = types.StringValue(snapshot.Name)
	c.VolumeID = types.Int64Value(int64(snapshot.Volume.ID))
	c.Size = types.Int64Value(int64(snapshot.Size))
	c.CreatedAt = types.StringValue(snapshot.CreatedAt.String())
}

func (c computeSnapshotDataSourceData) AppliesTo(snapshot compute.Snapshot) bool {
	if !c.ID.IsNull() && c.ID.ValueInt64() != int64(snapshot.ID) {
		return false
	}

	if !c.Name.IsNull() && c.Name.ValueString() != snapshot.Name {
		return false
	}

	if !c.VolumeID.IsNull() && c.VolumeID.ValueInt64() != int64(snapshot.Volume.ID) {
		return false
	}

	return true
}

func (c computeSnapshotDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the snapshot",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the snapshot",
				Optional:            true,
				Computed:            true,
			},
			"volume_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the volume",
				Optional:            true,
				Computed:            true,
			},
			"size": schema.Int64Attribute{
				MarkdownDescription: "size of the snapshot in GiB",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "date and time when the snapshot was created",
				Computed:            true,
			},
		},
	}
}

func newComputeSnapshotDataSource() datasource.DataSource {
	return &computeSnapshotDataSource{}
}

func (c *computeSnapshotDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_snapshot"
}

func (c *computeSnapshotDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.client = client
}

type computeSnapshotDataSource struct {
	client flowClient
}

func (c computeSnapshotDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var config computeSnapshotDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := c.client.Compute.Snapshot.List(ctx, core.CursorAll)
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
