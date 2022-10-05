package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/common"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/flowswiss/terraform-provider-flow/filter"
)

var _ tfsdk.DataSourceType = (*productDataSourceType)(nil)
var _ tfsdk.DataSource = (*productDataSource)(nil)

type productDataSourceData struct {
	ID   types.Int64  `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Type types.String `tfsdk:"type"`
}

func (p *productDataSourceData) FromEntity(product common.Product) {
	p.ID = types.Int64{Value: int64(product.ID)}
	p.Name = types.String{Value: product.Name}
	p.Type = types.String{Value: product.Type.Key}
}

func (p productDataSourceData) AppliesTo(product common.Product) bool {
	if !p.ID.Null && product.ID != int(p.ID.Value) {
		return false
	}

	if !p.Name.Null && product.Name != p.Name.Value {
		return false
	}

	if !p.Type.Null && product.Type.Key != p.Type.Value {
		return false
	}

	return true
}

type productDataSourceType struct{}

func (productDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the product",
				Optional:            true,
				Computed:            true,
			},
			"name": {
				Type:                types.StringType,
				MarkdownDescription: "name of the product",
				Optional:            true,
				Computed:            true,
			},
			"type": {
				Type:                types.StringType,
				MarkdownDescription: "type of the product",
				Optional:            true,
				Computed:            true,
			},
		},
	}, nil
}

func (productDataSourceType) NewDataSource(ctx context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return productDataSource{
		productService: common.NewProductService(prov.client),
	}, diagnostics
}

type productDataSource struct {
	productService common.ProductService
}

func (p productDataSource) Read(ctx context.Context, request tfsdk.ReadDataSourceRequest, response *tfsdk.ReadDataSourceResponse) {
	var config productDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := p.productService.List(ctx, goclient.Cursor{NoFilter: 1})
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
