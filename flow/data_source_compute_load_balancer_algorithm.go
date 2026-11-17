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
	_ datasource.DataSource              = (*computeLoadBalancerAlgorithmDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*computeLoadBalancerAlgorithmDataSource)(nil)
)

type computeLoadBalancerAlgorithmDataSourceData struct {
	ID   types.Int64  `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Key  types.String `tfsdk:"key"`
}

func (c *computeLoadBalancerAlgorithmDataSourceData) FromEntity(algorithm compute.LoadBalancerAlgorithm) {
	c.ID = types.Int64Value(int64(algorithm.ID))
	c.Name = types.StringValue(algorithm.Name)
	c.Key = types.StringValue(algorithm.Key)
}

func (c computeLoadBalancerAlgorithmDataSourceData) AppliesTo(algorithm compute.LoadBalancerAlgorithm) bool {
	if !c.ID.IsNull() && int(c.ID.ValueInt64()) != algorithm.ID {
		return false
	}

	if !c.Name.IsNull() && c.Name.ValueString() != algorithm.Name {
		return false
	}

	if !c.Key.IsNull() && c.Key.ValueString() != algorithm.Key {
		return false
	}

	return true
}

func (c computeLoadBalancerAlgorithmDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the load balancer algorithm",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the load balancer algorithm",
				Optional:            true,
				Computed:            true,
			},
			"key": schema.StringAttribute{
				MarkdownDescription: "unique key of the load balancer algorithm",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

func newComputeLoadBalancerAlgorithmDataSource() datasource.DataSource {
	return &computeLoadBalancerAlgorithmDataSource{}
}

func (c *computeLoadBalancerAlgorithmDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_load_balancer_algorithm"
}

func (c *computeLoadBalancerAlgorithmDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.client = client
}

type computeLoadBalancerAlgorithmDataSource struct {
	client flowClient
}

func (c computeLoadBalancerAlgorithmDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var config computeLoadBalancerAlgorithmDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := c.client.Compute.LoadBalancerEntity.ListAlgorithms(ctx, core.CursorAll)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list load balancer algorithms: %s", err))
		return
	}

	algorithm, err := filter.FindOne(config, list.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find load balancer algorithm: %s", err))
		return
	}

	var state computeLoadBalancerAlgorithmDataSourceData
	state.FromEntity(algorithm)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
