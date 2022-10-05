package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/common"
	"github.com/flowswiss/goclient/compute"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/flowswiss/terraform-provider-flow/filter"
)

var (
	_ tfsdk.DataSourceType = (*computeServerDataSourceType)(nil)
	_ tfsdk.DataSource     = (*computeServerDataSource)(nil)
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
	c.ID = types.Int64{Value: int64(server.ID)}
	c.Name = types.String{Value: server.Name}
	c.LocationID = types.Int64{Value: int64(server.Location.ID)}
	c.ImageID = types.Int64{Value: int64(server.Image.ID)}
	c.ProductID = types.Int64{Value: int64(server.Product.ID)}
	c.KeyPairID = types.Int64{Value: int64(server.KeyPair.ID)}
}

func (c computeServerDataSourceData) AppliesTo(server compute.Server) bool {
	if !c.ID.Null && c.ID.Value != int64(server.ID) {
		return false
	}

	if !c.Name.Null && c.Name.Value != server.Name {
		return false
	}

	if !c.LocationID.Null && c.LocationID.Value != int64(server.Location.ID) {
		return false
	}

	if !c.ImageID.Null && c.ImageID.Value != int64(server.Image.ID) {
		return false
	}

	if !c.ProductID.Null && c.ProductID.Value != int64(server.Product.ID) {
		return false
	}

	if !c.KeyPairID.Null && c.KeyPairID.Value != int64(server.KeyPair.ID) {
		return false
	}

	return true
}

type computeServerDataSourceType struct{}

func (c computeServerDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the server",
				Optional:            true,
				Computed:            true,
			},
			"name": {
				Type:                types.StringType,
				MarkdownDescription: "name of the server",
				Optional:            true,
				Computed:            true,
			},
			"location_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the location",
				Optional:            true,
				Computed:            true,
			},
			"image_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the image",
				Optional:            true,
				Computed:            true,
			},
			"product_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the product",
				Optional:            true,
				Computed:            true,
			},
			"key_pair_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the key pair",
				Optional:            true,
				Computed:            true,
			},
		},
	}, nil
}

func (c computeServerDataSourceType) NewDataSource(ctx context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return computeServerDataSource{
		serverService: compute.NewServerService(prov.client),
		orderService:  common.NewOrderService(prov.client),
	}, diagnostics
}

type computeServerDataSource struct {
	serverService compute.ServerService
	orderService  common.OrderService
}

func (c computeServerDataSource) Read(ctx context.Context, request tfsdk.ReadDataSourceRequest, response *tfsdk.ReadDataSourceResponse) {
	var config computeServerDataSourceData
	response.Diagnostics.Append(request.Config.Get(ctx, &config)...)
	if response.Diagnostics.HasError() {
		return
	}

	servers, err := c.serverService.List(ctx, goclient.Cursor{NoFilter: 1})
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
