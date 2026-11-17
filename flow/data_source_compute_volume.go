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
	_ datasource.DataSource              = (*computeVolumeDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*computeVolumeDataSource)(nil)
)

type computeVolumeDataSourceData struct {
	ID           types.Int64  `tfsdk:"id"`
	SerialNumber types.String `tfsdk:"serial_number"`
	Name         types.String `tfsdk:"name"`
	Size         types.Int64  `tfsdk:"size"`
	LocationID   types.Int64  `tfsdk:"location_id"`
}

func (c *computeVolumeDataSourceData) FromEntity(volume compute.Volume) {
	c.ID = types.Int64Value(int64(volume.ID))
	c.SerialNumber = types.StringValue(volume.SerialNumber)
	c.Name = types.StringValue(volume.Name)
	c.Size = types.Int64Value(int64(volume.Size))
	c.LocationID = types.Int64Value(int64(volume.Location.ID))
}

func (c computeVolumeDataSourceData) AppliesTo(volume compute.Volume) bool {
	if !c.ID.IsNull() && c.ID.ValueInt64() != int64(volume.ID) {
		return false
	}

	if !c.SerialNumber.IsNull() && c.SerialNumber.ValueString() != volume.SerialNumber {
		return false
	}

	if !c.Name.IsNull() && c.Name.ValueString() != volume.Name {
		return false
	}

	if !c.LocationID.IsNull() && c.LocationID.ValueInt64() != int64(volume.Location.ID) {
		return false
	}

	return true
}

func (c computeVolumeDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the volume",
				Optional:            true,
				Computed:            true,
			},
			"serial_number": schema.StringAttribute{
				MarkdownDescription: "unique serial number of the volume",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the volume",
				Optional:            true,
				Computed:            true,
			},
			"size": schema.Int64Attribute{
				MarkdownDescription: "size in GiB of the volume",
				Computed:            true,
			},
			"location_id": schema.Int64Attribute{
				MarkdownDescription: "identifier of the location of the volume",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

func newComputeVolumeDataSource() datasource.DataSource {
	return &computeVolumeDataSource{}
}

func (c *computeVolumeDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_volume"
}

func (c *computeVolumeDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.client = client
}

type computeVolumeDataSource struct {
	client flowClient
}

func (c computeVolumeDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var config computeVolumeDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := c.client.Compute.Volume.List(ctx, core.CursorAll)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list volumes: %s", err))
		return
	}

	volume, err := filter.FindOne(config, list.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find volume: %s", err))
		return
	}

	// the list endpoint leaves out the serial number
	volume, err = c.client.Compute.Volume.Get(ctx, compute.VolumeGetReq{ID: uint(volume.ID)})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get volume: %s", err))
		return
	}

	var state computeVolumeDataSourceData
	state.FromEntity(volume)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
