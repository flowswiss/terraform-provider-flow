package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/compute"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ tfsdk.DataSourceType = (*computeRouterDataSourceType)(nil)
	_ tfsdk.DataSource     = (*computeRouterDataSource)(nil)
)

type computeRouterDataSourceData struct {
	ID         types.Int64  `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	LocationID types.Int64  `tfsdk:"location_id"`
	Public     types.Bool   `tfsdk:"public"`
	PublicIP   types.String `tfsdk:"public_ip"`
}

func (c *computeRouterDataSourceData) FromEntity(router compute.Router) {
	c.ID = types.Int64{Value: int64(router.ID)}
	c.Name = types.String{Value: router.Name}
	c.LocationID = types.Int64{Value: int64(router.Location.ID)}
	c.Public = types.Bool{Value: router.Public}
}

type computeRouterDataSourceType struct{}

func (c computeRouterDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the router",
				Optional:            true,
				Computed:            true,
			},
			"name": {
				Type:                types.StringType,
				MarkdownDescription: "name of the router",
				Optional:            true,
				Computed:            true,
			},
			"location_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the location",
				Computed:            true,
			},
			"public": {
				Type:                types.BoolType,
				MarkdownDescription: "if the router is be public",
				Computed:            true,
			},
			"public_ip": {
				Type:                types.StringType,
				MarkdownDescription: "public IP of the router",
				Computed:            true,
			},
		},
	}, nil
}

func (c computeRouterDataSourceType) NewDataSource(ctx context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return computeRouterDataSource{
		routerService: compute.NewRouterService(prov.client),
	}, diagnostics
}

type computeRouterDataSource struct {
	routerService compute.RouterService
}

func (c computeRouterDataSource) Read(ctx context.Context, request tfsdk.ReadDataSourceRequest, response *tfsdk.ReadDataSourceResponse) {
	var config computeRouterDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := c.routerService.List(ctx, goclient.Cursor{NoFilter: 1})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list routers: %s", err))
		return
	}

	for _, router := range list.Items {
		if !config.ID.Null && int(config.ID.Value) != router.ID {
			continue
		}

		if !config.Name.Null && config.Name.Value != router.Name {
			continue
		}

		var state computeRouterDataSourceData
		state.FromEntity(router)

		diagnostics = response.State.Set(ctx, state)
		response.Diagnostics.Append(diagnostics...)
		return
	}

	response.Diagnostics.AddError("Not Found", "requested router could not be found")
}
