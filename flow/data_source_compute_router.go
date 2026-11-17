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
	_ datasource.DataSource              = (*computeRouterDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*computeRouterDataSource)(nil)
)

type computeRouterDataSourceData struct {
	ID         types.Int64  `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	LocationID types.Int64  `tfsdk:"location_id"`
	Public     types.Bool   `tfsdk:"public"`
	PublicIP   types.String `tfsdk:"public_ip"`
}

func (c *computeRouterDataSourceData) FromEntity(router compute.Router) {
	c.ID = types.Int64Value(int64(router.ID))
	c.Name = types.StringValue(router.Name)
	c.LocationID = types.Int64Value(int64(router.Location.ID))
	c.Public = types.BoolValue(router.Public)

	c.PublicIP = types.StringNull()
	if router.Public {
		c.PublicIP = types.StringValue(router.PublicIP)
	}
}

func (c computeRouterDataSourceData) AppliesTo(router compute.Router) bool {
	if !c.ID.IsNull() && int(c.ID.ValueInt64()) != router.ID {
		return false
	}

	if !c.Name.IsNull() && c.Name.ValueString() != router.Name {
		return false
	}

	return true
}

func (c computeRouterDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the router",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the router",
				Optional:            true,
				Computed:            true,
			},
			"location_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the location",
				Computed:            true,
			},
			"public": schema.BoolAttribute{
				MarkdownDescription: "if the router is be public",
				Computed:            true,
			},
			"public_ip": schema.StringAttribute{
				MarkdownDescription: "public IP of the router",
				Computed:            true,
			},
		},
	}
}

func newComputeRouterDataSource() datasource.DataSource {
	return &computeRouterDataSource{}
}

func (c *computeRouterDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_router"
}

func (c *computeRouterDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.client = client
}

type computeRouterDataSource struct {
	client flowClient
}

func (c computeRouterDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var config computeRouterDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := c.client.Compute.Router.List(ctx, core.CursorAll)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list routers: %s", err))
		return
	}

	router, err := filter.FindOne(config, list.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find router: %s", err))
		return
	}

	var state computeRouterDataSourceData
	state.FromEntity(router)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
