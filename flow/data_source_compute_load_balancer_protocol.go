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
	_ datasource.DataSource              = (*computeLoadBalancerProtocolDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*computeLoadBalancerProtocolDataSource)(nil)
)

type computeLoadBalancerProtocolDataSourceData struct {
	ID   types.Int64  `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Key  types.String `tfsdk:"key"`
}

func (c *computeLoadBalancerProtocolDataSourceData) FromEntity(protocol compute.LoadBalancerProtocol) {
	c.ID = types.Int64Value(int64(protocol.ID))
	c.Name = types.StringValue(protocol.Name)
	c.Key = types.StringValue(protocol.Key)
}

func (c computeLoadBalancerProtocolDataSourceData) AppliesTo(protocol compute.LoadBalancerProtocol) bool {
	if !c.ID.IsNull() && int(c.ID.ValueInt64()) != protocol.ID {
		return false
	}

	if !c.Name.IsNull() && c.Name.ValueString() != protocol.Name {
		return false
	}

	if !c.Key.IsNull() && c.Key.ValueString() != protocol.Key {
		return false
	}

	return true
}

func (c computeLoadBalancerProtocolDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the load balancer protocol",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the load balancer protocol",
				Optional:            true,
				Computed:            true,
			},
			"key": schema.StringAttribute{
				MarkdownDescription: "unique key of the load balancer protocol",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

func newComputeLoadBalancerProtocolDataSource() datasource.DataSource {
	return &computeLoadBalancerProtocolDataSource{}
}

func (c *computeLoadBalancerProtocolDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_load_balancer_protocol"
}

func (c *computeLoadBalancerProtocolDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.client = client
}

type computeLoadBalancerProtocolDataSource struct {
	client flowClient
}

func (c computeLoadBalancerProtocolDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var config computeLoadBalancerProtocolDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := c.client.Compute.LoadBalancerEntity.ListProtocols(ctx, core.CursorAll)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list load balancer protocols: %s", err))
		return
	}

	protocol, err := filter.FindOne(config, list.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find load balancer protocol: %s", err))
		return
	}

	var state computeLoadBalancerProtocolDataSourceData
	state.FromEntity(protocol)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
