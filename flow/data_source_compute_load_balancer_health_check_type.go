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
	_ datasource.DataSource              = (*computeLoadBalancerHealthCheckTypeDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*computeLoadBalancerHealthCheckTypeDataSource)(nil)
)

type computeLoadBalancerHealthCheckTypeDataSourceData struct {
	ID   types.Int64  `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Key  types.String `tfsdk:"key"`
}

func (c *computeLoadBalancerHealthCheckTypeDataSourceData) FromEntity(healthCheckType compute.LoadBalancerHealthCheckType) {
	c.ID = types.Int64Value(int64(healthCheckType.ID))
	c.Name = types.StringValue(healthCheckType.Name)
	c.Key = types.StringValue(healthCheckType.Key)
}

func (c computeLoadBalancerHealthCheckTypeDataSourceData) AppliesTo(healthCheckType compute.LoadBalancerHealthCheckType) bool {
	if !c.ID.IsNull() && int(c.ID.ValueInt64()) != healthCheckType.ID {
		return false
	}

	if !c.Name.IsNull() && c.Name.ValueString() != healthCheckType.Name {
		return false
	}

	if !c.Key.IsNull() && c.Key.ValueString() != healthCheckType.Key {
		return false
	}

	return true
}

func (c computeLoadBalancerHealthCheckTypeDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the load balancer health check type",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the load balancer health check type",
				Optional:            true,
				Computed:            true,
			},
			"key": schema.StringAttribute{
				MarkdownDescription: "unique key of the load balancer health check type",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

func newComputeLoadBalancerHealthCheckTypeDataSource() datasource.DataSource {
	return &computeLoadBalancerHealthCheckTypeDataSource{}
}

func (c *computeLoadBalancerHealthCheckTypeDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_load_balancer_health_check_type"
}

func (c *computeLoadBalancerHealthCheckTypeDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.client = client
}

type computeLoadBalancerHealthCheckTypeDataSource struct {
	client flowClient
}

func (c computeLoadBalancerHealthCheckTypeDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var config computeLoadBalancerHealthCheckTypeDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := c.client.Compute.LoadBalancerEntity.ListHealthCheckTypes(ctx, core.CursorAll)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list load balancer health check types: %s", err))
		return
	}

	healthCheckType, err := filter.FindOne(config, list.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find load balancer health check type: %s", err))
		return
	}

	var state computeLoadBalancerHealthCheckTypeDataSourceData
	state.FromEntity(healthCheckType)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
