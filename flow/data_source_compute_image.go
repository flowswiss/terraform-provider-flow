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

var _ datasource.DataSource = (*computeImageDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*computeImageDataSource)(nil)

type computeImageDataSourceData struct {
	ID              types.Int64  `tfsdk:"id"`
	OperatingSystem types.String `tfsdk:"operating_system"`
	Version         types.String `tfsdk:"version"`
	Key             types.String `tfsdk:"key"`
	Category        types.String `tfsdk:"category"`
	Type            types.String `tfsdk:"type"`
	Username        types.String `tfsdk:"username"`
	MinRootDiskSize types.Int64  `tfsdk:"min_root_disk_size"`
}

func (i *computeImageDataSourceData) FromEntity(image compute.Image) {
	i.ID = types.Int64Value(int64(image.ID))
	i.OperatingSystem = types.StringValue(image.OperatingSystem)
	i.Version = types.StringValue(image.Version)
	i.Key = types.StringValue(image.Key)
	i.Category = types.StringValue(image.Category)
	i.Type = types.StringValue(image.Type)
	i.Username = types.StringValue(image.Username)
	i.MinRootDiskSize = types.Int64Value(int64(image.MinRootDiskSize))
}

func (i computeImageDataSourceData) AppliesTo(image compute.Image) bool {
	if !i.ID.IsNull() && image.ID != int(i.ID.ValueInt64()) {
		return false
	}

	if !i.OperatingSystem.IsNull() && image.OperatingSystem != i.OperatingSystem.ValueString() {
		return false
	}

	if !i.Version.IsNull() && image.Version != i.Version.ValueString() {
		return false
	}

	if !i.Key.IsNull() && image.Key != i.Key.ValueString() {
		return false

	}

	if !i.Category.IsNull() && image.Category != i.Category.ValueString() {
		return false
	}

	if !i.Type.IsNull() && image.Type != i.Type.ValueString() {
		return false
	}

	return true
}

func (computeImageDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the image",
				Optional:            true,
				Computed:            true,
			},
			"operating_system": schema.StringAttribute{
				MarkdownDescription: "operating system of the image",
				Optional:            true,
				Computed:            true,
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "version of the image",
				Optional:            true,
				Computed:            true,
			},
			"key": schema.StringAttribute{
				MarkdownDescription: "unique key of the image",
				Optional:            true,
				Computed:            true,
			},
			"category": schema.StringAttribute{
				MarkdownDescription: "category of the image (e.g. 'linux', 'windows')",
				Optional:            true,
				Computed:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "type of the image",
				Optional:            true,
				Computed:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "default username to connect to the server with",
				Computed:            true,
			},
			"min_root_disk_size": schema.Int64Attribute{
				MarkdownDescription: "minimum root disk size for servers using this image",
				Computed:            true,
			},
		},
	}
}

func newComputeImageDataSource() datasource.DataSource {
	return &computeImageDataSource{}
}

func (i *computeImageDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_image"
}

func (i *computeImageDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	i.client = client
}

type computeImageDataSource struct {
	client flowClient
}

func (i computeImageDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var config computeImageDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := i.client.Compute.Image.List(ctx, core.CursorAll)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get images: %s", err))
		return
	}

	image, err := filter.FindOne(config, list.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find image: %s", err))
		return
	}

	var state computeImageDataSourceData
	state.FromEntity(image)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
