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
	_ datasource.DataSource              = (*computeElasticIPDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*computeElasticIPDataSource)(nil)
)

type computeElasticIPDataSourceData struct {
	ID         types.Int64  `tfsdk:"id"`
	LocationID types.Int64  `tfsdk:"location_id"`
	PublicIP   types.String `tfsdk:"public_ip"`
}

func (c *computeElasticIPDataSourceData) FromEntity(elasticIP compute.ElasticIP) {
	c.ID = types.Int64Value(int64(elasticIP.ID))
	c.LocationID = types.Int64Value(int64(elasticIP.Location.ID))
	c.PublicIP = types.StringValue(elasticIP.PublicIP)
}

func (c computeElasticIPDataSourceData) AppliesTo(elasticIP compute.ElasticIP) bool {
	if !c.ID.IsNull() && c.ID.ValueInt64() != int64(elasticIP.ID) {
		return false
	}

	if !c.LocationID.IsNull() && c.LocationID.ValueInt64() != int64(elasticIP.Location.ID) {
		return false
	}

	if !c.PublicIP.IsNull() && c.PublicIP.ValueString() != elasticIP.PublicIP {
		return false
	}

	return true
}

func (c computeElasticIPDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the elastic ip",
				Optional:            true,
				Computed:            true,
			},
			"location_id": schema.Int64Attribute{
				MarkdownDescription: "location of the elastic ip",
				Optional:            true,
				Computed:            true,
			},
			"public_ip": schema.StringAttribute{
				MarkdownDescription: "public ip address",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

func newComputeElasticIPDataSource() datasource.DataSource {
	return &computeElasticIPDataSource{}
}

func (c *computeElasticIPDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_elastic_ip"
}

func (c *computeElasticIPDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.client = client
}

type computeElasticIPDataSource struct {
	client flowClient
}

func (c computeElasticIPDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var config computeElasticIPDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := c.client.Compute.ElasticIP.List(ctx, core.CursorAll)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list elastic ips: %s", err))
		return
	}

	elasticIP, err := filter.FindOne(config, list.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find elastic ip: %s", err))
		return
	}

	var state computeElasticIPDataSourceData
	state.FromEntity(elasticIP)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
