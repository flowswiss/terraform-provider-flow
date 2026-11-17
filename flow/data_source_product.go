package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/v2/common"
	"github.com/flowswiss/goclient/v2/core"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/flowswiss/terraform-provider-flow/filter"
)

var _ datasource.DataSource = (*productDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*productDataSource)(nil)

type productDataSourceData struct {
	ID   types.Int64  `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Type types.String `tfsdk:"type"`
}

func (p *productDataSourceData) FromEntity(product common.Product) {
	p.ID = types.Int64Value(int64(product.ID))
	p.Name = types.StringValue(product.Name)
	p.Type = types.StringValue(product.Type.Key)
}

func (p productDataSourceData) AppliesTo(product common.Product) bool {
	if !p.ID.IsNull() && product.ID != int(p.ID.ValueInt64()) {
		return false
	}

	if !p.Name.IsNull() && product.Name != p.Name.ValueString() {
		return false
	}

	if !p.Type.IsNull() && product.Type.Key != p.Type.ValueString() {
		return false
	}

	return true
}

func (productDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the product",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the product",
				Optional:            true,
				Computed:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "type of the product",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

func newProductDataSource() datasource.DataSource {
	return &productDataSource{}
}

func (p *productDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_product"
}

func (p *productDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	p.client = client
}

type productDataSource struct {
	client flowClient
}

func (p productDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var config productDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := p.client.Common.Product.List(ctx, core.CursorAll)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get products: %s", err))
		return
	}

	product, err := filter.FindOne(config, list.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find product: %s", err))
		return
	}

	var state productDataSourceData
	state.FromEntity(product)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
