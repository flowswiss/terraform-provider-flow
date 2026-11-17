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
	_ datasource.DataSource              = (*computeServerDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*computeServerDataSource)(nil)
)

type computeServerDataSourceData struct {
	ID         types.Int64  `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	LocationID types.Int64  `tfsdk:"location_id"`
	ImageID    types.Int64  `tfsdk:"image_id"`
	ProductID  types.Int64  `tfsdk:"product_id"`
	KeyPairID  types.Int64  `tfsdk:"key_pair_id"`
}

func (c *computeServerDataSourceData) FromEntity(server compute.Server) {
	c.ID = types.Int64Value(int64(server.ID))
	c.Name = types.StringValue(server.Name)
	c.LocationID = types.Int64Value(int64(server.Location.ID))
	c.ImageID = types.Int64Value(int64(server.Image.ID))
	c.ProductID = types.Int64Value(int64(server.Product.ID))
	c.KeyPairID = types.Int64Value(int64(server.KeyPair.ID))
}

func (c computeServerDataSourceData) AppliesTo(server compute.Server) bool {
	if !c.ID.IsNull() && c.ID.ValueInt64() != int64(server.ID) {
		return false
	}

	if !c.Name.IsNull() && c.Name.ValueString() != server.Name {
		return false
	}

	if !c.LocationID.IsNull() && c.LocationID.ValueInt64() != int64(server.Location.ID) {
		return false
	}

	if !c.ImageID.IsNull() && c.ImageID.ValueInt64() != int64(server.Image.ID) {
		return false
	}

	if !c.ProductID.IsNull() && c.ProductID.ValueInt64() != int64(server.Product.ID) {
		return false
	}

	if !c.KeyPairID.IsNull() && c.KeyPairID.ValueInt64() != int64(server.KeyPair.ID) {
		return false
	}

	return true
}

func (c computeServerDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the server",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the server",
				Optional:            true,
				Computed:            true,
			},
			"location_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the location",
				Optional:            true,
				Computed:            true,
			},
			"image_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the image",
				Optional:            true,
				Computed:            true,
			},
			"product_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the product",
				Optional:            true,
				Computed:            true,
			},
			"key_pair_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the key pair",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

func newComputeServerDataSource() datasource.DataSource {
	return &computeServerDataSource{}
}

func (c *computeServerDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_server"
}

func (c *computeServerDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.client = client
}

type computeServerDataSource struct {
	client flowClient
}

func (c computeServerDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var config computeServerDataSourceData
	response.Diagnostics.Append(request.Config.Get(ctx, &config)...)
	if response.Diagnostics.HasError() {
		return
	}

	servers, err := c.client.Compute.Server.List(ctx, core.CursorAll)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get server: %s", err))
		return
	}

	server, err := filter.FindOne(config, servers.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find server: %s", err))
		return
	}

	var state computeServerDataSourceData
	state.FromEntity(server)
	response.Diagnostics.Append(response.State.Set(ctx, state)...)
}
