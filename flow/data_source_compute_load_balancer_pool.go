package flow

import (
	"context"
	"fmt"
	"time"

	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/compute"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/flowswiss/terraform-provider-flow/filter"
)

var (
	_ tfsdk.DataSourceType = (*computeLoadBalancerPoolDataSourceType)(nil)
	_ tfsdk.DataSource     = (*computeLoadBalancerPoolDataSource)(nil)
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
	c.ID = types.Int64{Value: int64(pool.ID)}
	c.LoadBalancerID = types.Int64{Value: int64(loadBalancerID)}
	c.Name = types.String{Value: pool.Name}

	c.BalancingAlgorithmID = types.Int64{Value: int64(pool.Algorithm.ID)}
	c.StickySession = types.Bool{Value: pool.StickySession}

	c.EntryProtocolID = types.Int64{Value: int64(pool.EntryProtocol.ID)}
	c.EntryPort = types.Int64{Value: int64(pool.EntryPort)}
	c.TargetProtocolID = types.Int64{Value: int64(pool.TargetProtocol.ID)}

	if pool.Certificate.ID == 0 {
		c.CertificateID = types.Int64{Null: true}
	} else {
		c.CertificateID = types.Int64{Value: int64(pool.Certificate.ID)}
	}

	c.HealthCheck = &computeLoadBalancerHealthCheckDataSourceData{
		TypeID:             types.Int64{Value: int64(pool.HealthCheck.Type.ID)},
		HTTP:               nil,
		Interval:           types.String{Value: (time.Duration(pool.HealthCheck.Interval) * time.Second).String()},
		Timeout:            types.String{Value: (time.Duration(pool.HealthCheck.Timeout) * time.Second).String()},
		HealthyThreshold:   types.Int64{Value: int64(pool.HealthCheck.HealthyThreshold)},
		UnhealthyThreshold: types.Int64{Value: int64(pool.HealthCheck.UnhealthyThreshold)},
	}

	if pool.HealthCheck.HTTPMethod != "" || pool.HealthCheck.HTTPPath != "" {
		c.HealthCheck.HTTP = &computeLoadBalancerHTTPHealthCheckDataSourceData{
			Method: types.String{Value: pool.HealthCheck.HTTPMethod},
			Path:   types.String{Value: pool.HealthCheck.HTTPPath},
		}
	}
}

func (c computeLoadBalancerPoolDataSourceData) AppliesTo(pool compute.LoadBalancerPool) bool {
	if !c.ID.Null && c.ID.Value != int64(pool.ID) {
		return false
	}

	if !c.BalancingAlgorithmID.Null && c.BalancingAlgorithmID.Value != int64(pool.Algorithm.ID) {
		return false
	}

	if !c.EntryProtocolID.Null && c.EntryProtocolID.Value != int64(pool.EntryProtocol.ID) {
		return false
	}

	if !c.EntryPort.Null && c.EntryPort.Value != int64(pool.EntryPort) {
		return false
	}

	if !c.TargetProtocolID.Null && c.TargetProtocolID.Value != int64(pool.TargetProtocol.ID) {
		return false
	}

	return true
}

type computeLoadBalancerPoolDataSourceType struct{}

func (c computeLoadBalancerPoolDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the load balancer pool",
				Optional:            true,
				Computed:            true,
			},
			"load_balancer_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the load balancer",
				Required:            true,
			},

			"name": {
				Type:                types.StringType,
				MarkdownDescription: "name of the load balancer pool",
				Computed:            true,
			},

			"balancing_algorithm_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the balancing algorithm",
				Optional:            true,
				Computed:            true,
			},
			"sticky_session": {
				Type:                types.BoolType,
				MarkdownDescription: "whether the load balancer pool is sticky",
				Computed:            true,
			},

			"entry_protocol_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the entry protocol",
				Optional:            true,
				Computed:            true,
			},
			"entry_port": {
				Type:                types.Int64Type,
				MarkdownDescription: "entry port of the load balancer pool",
				Optional:            true,
				Computed:            true,
			},
			"target_protocol_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the target protocol",
				Optional:            true,
				Computed:            true,
			},

			"certificate_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the certificate",
				Computed:            true,
			},

			"health_check": {
				Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
					"type_id": {
						Type:                types.Int64Type,
						MarkdownDescription: "unique identifier of the health check type",
						Computed:            true,
					},
					"http": {
						Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
							"method": {
								Type:                types.StringType,
								MarkdownDescription: "HTTP method of the health check",
								Computed:            true,
							},
							"path": {
								Type:                types.StringType,
								MarkdownDescription: "path of the health check",
								Computed:            true,
							},
						}),
						Computed: true,
					},
					"interval": {
						Type:                types.StringType,
						MarkdownDescription: "interval duration of the health check",
						Computed:            true,
					},
					"timeout": {
						Type:                types.StringType,
						MarkdownDescription: "timeout duration of the health check",
						Computed:            true,
					},
					"healthy_threshold": {
						Type:                types.Int64Type,
						MarkdownDescription: "number of successful health checks before considering the target healthy",
						Computed:            true,
					},
					"unhealthy_threshold": {
						Type:                types.Int64Type,
						MarkdownDescription: "number of failed health checks before considering the target unhealthy",
						Computed:            true,
					},
				}),
				Computed: true,
			},
		},
	}, nil
}

func (c computeLoadBalancerPoolDataSourceType) NewDataSource(ctx context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return computeLoadBalancerPoolDataSource{
		loadBalancerService: compute.NewLoadBalancerService(prov.client),
	}, diagnostics
}

type computeLoadBalancerPoolDataSource struct {
	loadBalancerService compute.LoadBalancerService
}

func (c computeLoadBalancerPoolDataSource) Read(ctx context.Context, request tfsdk.ReadDataSourceRequest, response *tfsdk.ReadDataSourceResponse) {
	var config computeLoadBalancerPoolDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	loadBalancerID := int(config.LoadBalancerID.Value)

	list, err := c.loadBalancerService.Pools(loadBalancerID).List(ctx, goclient.Cursor{NoFilter: 1})
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
