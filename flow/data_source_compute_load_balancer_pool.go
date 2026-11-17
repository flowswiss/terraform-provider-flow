package flow

import (
	"context"
	"fmt"
	"time"

	"github.com/flowswiss/goclient/v2/compute"
	"github.com/flowswiss/goclient/v2/core"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/flowswiss/terraform-provider-flow/filter"
)

var (
	_ datasource.DataSource              = (*computeLoadBalancerPoolDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*computeLoadBalancerPoolDataSource)(nil)
)

type computeLoadBalancerHTTPHealthCheckDataSourceData struct {
	Method types.String `tfsdk:"method"`
	Path   types.String `tfsdk:"path"`
}

type computeLoadBalancerHealthCheckDataSourceData struct {
	TypeID types.Int64                                       `tfsdk:"type_id"`
	HTTP   *computeLoadBalancerHTTPHealthCheckDataSourceData `tfsdk:"http"`

	Interval types.String `tfsdk:"interval"`
	Timeout  types.String `tfsdk:"timeout"`

	HealthyThreshold   types.Int64 `tfsdk:"healthy_threshold"`
	UnhealthyThreshold types.Int64 `tfsdk:"unhealthy_threshold"`
}

type computeLoadBalancerPoolDataSourceData struct {
	ID             types.Int64 `tfsdk:"id"`
	LoadBalancerID types.Int64 `tfsdk:"load_balancer_id"`

	Name types.String `tfsdk:"name"`

	BalancingAlgorithmID types.Int64 `tfsdk:"balancing_algorithm_id"`
	StickySession        types.Bool  `tfsdk:"sticky_session"`

	EntryProtocolID  types.Int64 `tfsdk:"entry_protocol_id"`
	EntryPort        types.Int64 `tfsdk:"entry_port"`
	TargetProtocolID types.Int64 `tfsdk:"target_protocol_id"`

	CertificateID types.Int64 `tfsdk:"certificate_id"`

	HealthCheck *computeLoadBalancerHealthCheckDataSourceData `tfsdk:"health_check"`
}

func (c *computeLoadBalancerPoolDataSourceData) FromEntity(loadBalancerID int, pool compute.LoadBalancerPool) {
	c.ID = types.Int64Value(int64(pool.ID))
	c.LoadBalancerID = types.Int64Value(int64(loadBalancerID))
	c.Name = types.StringValue(pool.Name)

	c.BalancingAlgorithmID = types.Int64Value(int64(pool.Algorithm.ID))
	c.StickySession = types.BoolValue(pool.StickySession)

	c.EntryProtocolID = types.Int64Value(int64(pool.EntryProtocol.ID))
	c.EntryPort = types.Int64Value(int64(pool.EntryPort))
	c.TargetProtocolID = types.Int64Value(int64(pool.TargetProtocol.ID))

	if pool.Certificate.ID == 0 {
		c.CertificateID = types.Int64Null()
	} else {
		c.CertificateID = types.Int64Value(int64(pool.Certificate.ID))
	}

	c.HealthCheck = &computeLoadBalancerHealthCheckDataSourceData{
		TypeID:             types.Int64Value(int64(pool.HealthCheck.Type.ID)),
		HTTP:               nil,
		Interval:           types.StringValue((time.Duration(pool.HealthCheck.Interval) * time.Second).String()),
		Timeout:            types.StringValue((time.Duration(pool.HealthCheck.Timeout) * time.Second).String()),
		HealthyThreshold:   types.Int64Value(int64(pool.HealthCheck.HealthyThreshold)),
		UnhealthyThreshold: types.Int64Value(int64(pool.HealthCheck.UnhealthyThreshold)),
	}

	if pool.HealthCheck.HTTPMethod != "" || pool.HealthCheck.HTTPPath != "" {
		c.HealthCheck.HTTP = &computeLoadBalancerHTTPHealthCheckDataSourceData{
			Method: types.StringValue(pool.HealthCheck.HTTPMethod),
			Path:   types.StringValue(pool.HealthCheck.HTTPPath),
		}
	}
}

func (c computeLoadBalancerPoolDataSourceData) AppliesTo(pool compute.LoadBalancerPool) bool {
	if !c.ID.IsNull() && c.ID.ValueInt64() != int64(pool.ID) {
		return false
	}

	if !c.BalancingAlgorithmID.IsNull() && c.BalancingAlgorithmID.ValueInt64() != int64(pool.Algorithm.ID) {
		return false
	}

	if !c.EntryProtocolID.IsNull() && c.EntryProtocolID.ValueInt64() != int64(pool.EntryProtocol.ID) {
		return false
	}

	if !c.EntryPort.IsNull() && c.EntryPort.ValueInt64() != int64(pool.EntryPort) {
		return false
	}

	if !c.TargetProtocolID.IsNull() && c.TargetProtocolID.ValueInt64() != int64(pool.TargetProtocol.ID) {
		return false
	}

	return true
}

func (c computeLoadBalancerPoolDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the load balancer pool",
				Optional:            true,
				Computed:            true,
			},
			"load_balancer_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the load balancer",
				Required:            true,
			},

			"name": schema.StringAttribute{
				MarkdownDescription: "name of the load balancer pool",
				Computed:            true,
			},

			"balancing_algorithm_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the balancing algorithm",
				Optional:            true,
				Computed:            true,
			},
			"sticky_session": schema.BoolAttribute{
				MarkdownDescription: "whether the load balancer pool is sticky",
				Computed:            true,
			},

			"entry_protocol_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the entry protocol",
				Optional:            true,
				Computed:            true,
			},
			"entry_port": schema.Int64Attribute{
				MarkdownDescription: "entry port of the load balancer pool",
				Optional:            true,
				Computed:            true,
			},
			"target_protocol_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the target protocol",
				Optional:            true,
				Computed:            true,
			},

			"certificate_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the certificate",
				Computed:            true,
			},

			"health_check": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"type_id": schema.Int64Attribute{
						MarkdownDescription: "unique identifier of the health check type",
						Computed:            true,
					},
					"http": schema.SingleNestedAttribute{
						Attributes: map[string]schema.Attribute{
							"method": schema.StringAttribute{
								MarkdownDescription: "HTTP method of the health check",
								Computed:            true,
							},
							"path": schema.StringAttribute{
								MarkdownDescription: "path of the health check",
								Computed:            true,
							},
						},
						Computed: true,
					},
					"interval": schema.StringAttribute{
						MarkdownDescription: "interval duration of the health check",
						Computed:            true,
					},
					"timeout": schema.StringAttribute{
						MarkdownDescription: "timeout duration of the health check",
						Computed:            true,
					},
					"healthy_threshold": schema.Int64Attribute{
						MarkdownDescription: "number of successful health checks before considering the target healthy",
						Computed:            true,
					},
					"unhealthy_threshold": schema.Int64Attribute{
						MarkdownDescription: "number of failed health checks before considering the target unhealthy",
						Computed:            true,
					},
				},
				Computed: true,
			},
		},
	}
}

func newComputeLoadBalancerPoolDataSource() datasource.DataSource {
	return &computeLoadBalancerPoolDataSource{}
}

func (c *computeLoadBalancerPoolDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_load_balancer_pool"
}

func (c *computeLoadBalancerPoolDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.client = client
}

type computeLoadBalancerPoolDataSource struct {
	client flowClient
}

func (c computeLoadBalancerPoolDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var config computeLoadBalancerPoolDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	loadBalancerID := int(config.LoadBalancerID.ValueInt64())

	list, err := c.client.Compute.LoadBalancerPool.List(ctx, compute.LoadBalancerPoolListReq{LoadBalancerID: uint(loadBalancerID), Cursor: core.CursorAll})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get load balancer pool: %s", err))
		return
	}

	pool, err := filter.FindOne(config, list.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find load balancer pool: %s", err))
		return
	}

	var state computeLoadBalancerPoolDataSourceData
	state.FromEntity(loadBalancerID, pool)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
