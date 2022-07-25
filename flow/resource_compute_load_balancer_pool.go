package flow

import (
	"context"
	"fmt"
	"time"

	"github.com/flowswiss/goclient/compute"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ tfsdk.ResourceType = (*computeLoadBalancerPoolResourceType)(nil)
	_ tfsdk.Resource     = (*computeLoadBalancerPoolResource)(nil)
)

type computeLoadBalancerHTTPHealthCheckResourceData struct {
	Method types.String `tfsdk:"method"`
	Path   types.String `tfsdk:"path"`
}

type computeLoadBalancerHealthCheckResourceData struct {
	TypeID types.Int64                                     `tfsdk:"type_id"`
	HTTP   *computeLoadBalancerHTTPHealthCheckResourceData `tfsdk:"http"`

	Interval types.String `tfsdk:"interval"`
	Timeout  types.String `tfsdk:"timeout"`

	HealthyThreshold   types.Int64 `tfsdk:"healthy_threshold"`
	UnhealthyThreshold types.Int64 `tfsdk:"unhealthy_threshold"`
}

type computeLoadBalancerPoolResourceData struct {
	ID             types.Int64 `tfsdk:"id"`
	LoadBalancerID types.Int64 `tfsdk:"load_balancer_id"`

	Name types.String `tfsdk:"name"`

	BalancingAlgorithmID types.Int64 `tfsdk:"balancing_algorithm_id"`
	StickySession        types.Bool  `tfsdk:"sticky_session"`

	EntryProtocolID  types.Int64 `tfsdk:"entry_protocol_id"`
	EntryPort        types.Int64 `tfsdk:"entry_port"`
	TargetProtocolID types.Int64 `tfsdk:"target_protocol_id"`

	CertificateID types.Int64 `tfsdk:"certificate_id"`

	HealthCheck *computeLoadBalancerHealthCheckResourceData `tfsdk:"health_check"`
}

func (c *computeLoadBalancerPoolResourceData) FromEntity(loadBalancerID int, pool compute.LoadBalancerPool) {
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

	c.HealthCheck = &computeLoadBalancerHealthCheckResourceData{
		TypeID:             types.Int64{Value: int64(pool.HealthCheck.Type.ID)},
		HTTP:               nil,
		Interval:           types.String{Value: (time.Duration(pool.HealthCheck.Interval) * time.Second).String()},
		Timeout:            types.String{Value: (time.Duration(pool.HealthCheck.Timeout) * time.Second).String()},
		HealthyThreshold:   types.Int64{Value: int64(pool.HealthCheck.HealthyThreshold)},
		UnhealthyThreshold: types.Int64{Value: int64(pool.HealthCheck.UnhealthyThreshold)},
	}

	if pool.HealthCheck.HTTPMethod != "" || pool.HealthCheck.HTTPPath != "" {
		c.HealthCheck.HTTP = &computeLoadBalancerHTTPHealthCheckResourceData{
			Method: types.String{Value: pool.HealthCheck.HTTPMethod},
			Path:   types.String{Value: pool.HealthCheck.HTTPPath},
		}
	}
}

type computeLoadBalancerPoolResourceType struct{}

func (c computeLoadBalancerPoolResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the load balancer pool",
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},
			"load_balancer_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the load balancer",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},

			"name": {
				Type:                types.StringType,
				MarkdownDescription: "name of the load balancer pool",
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},

			"balancing_algorithm_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the balancing algorithm",
				Required:            true,
			},
			"sticky_session": {
				Type:                types.BoolType,
				MarkdownDescription: "whether the load balancer pool is sticky",
				Optional:            true,
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},

			"entry_protocol_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the entry protocol",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"entry_port": {
				Type:                types.Int64Type,
				MarkdownDescription: "entry port of the load balancer pool",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"target_protocol_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the target protocol",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},

			"certificate_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the certificate",
				Optional:            true,
			},

			"health_check": {
				Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
					"type_id": {
						Type:                types.Int64Type,
						MarkdownDescription: "unique identifier of the health check type",
						Required:            true,
					},
					"http": {
						Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
							"method": {
								Type:                types.StringType,
								MarkdownDescription: "HTTP method of the health check",
								Required:            true,
							},
							"path": {
								Type:                types.StringType,
								MarkdownDescription: "path of the health check",
								Required:            true,
							},
						}),
						Optional: true,
						Computed: true,
						PlanModifiers: tfsdk.AttributePlanModifiers{
							tfsdk.UseStateForUnknown(),
						},
					},
					"interval": {
						Type:                types.StringType,
						MarkdownDescription: "interval duration of the health check",
						Optional:            true,
						Computed:            true,
						PlanModifiers: tfsdk.AttributePlanModifiers{
							tfsdk.UseStateForUnknown(),
						},
					},
					"timeout": {
						Type:                types.StringType,
						MarkdownDescription: "timeout duration of the health check",
						Optional:            true,
						Computed:            true,
						PlanModifiers: tfsdk.AttributePlanModifiers{
							tfsdk.UseStateForUnknown(),
						},
					},
					"healthy_threshold": {
						Type:                types.Int64Type,
						MarkdownDescription: "number of successful health checks before considering the target healthy",
						Optional:            true,
						Computed:            true,
						PlanModifiers: tfsdk.AttributePlanModifiers{
							tfsdk.UseStateForUnknown(),
						},
					},
					"unhealthy_threshold": {
						Type:                types.Int64Type,
						MarkdownDescription: "number of failed health checks before considering the target unhealthy",
						Optional:            true,
						Computed:            true,
						PlanModifiers: tfsdk.AttributePlanModifiers{
							tfsdk.UseStateForUnknown(),
						},
					},
				}),
				Required: true,
			},
		},
	}, nil
}

func (c computeLoadBalancerPoolResourceType) NewResource(ctx context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return computeLoadBalancerPoolResource{
		loadBalancerService: compute.NewLoadBalancerService(prov.client),
	}, diagnostics
}

type computeLoadBalancerPoolResource struct {
	loadBalancerService compute.LoadBalancerService
}

func (c computeLoadBalancerPoolResource) Create(ctx context.Context, request tfsdk.CreateResourceRequest, response *tfsdk.CreateResourceResponse) {
	var config computeLoadBalancerPoolResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	healthCheck, diagnostics := convertHealthCheckConfigToAPIOptions(*config.HealthCheck)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	loadBalancerID := int(config.LoadBalancerID.Value)

	create := compute.LoadBalancerPoolCreate{
		EntryProtocolID:      int(config.EntryProtocolID.Value),
		TargetProtocolID:     int(config.TargetProtocolID.Value),
		CertificateID:        int(config.CertificateID.Value),
		EntryPort:            int(config.EntryPort.Value),
		BalancingAlgorithmID: int(config.BalancingAlgorithmID.Value),
		StickySession:        config.StickySession.Value,
		HealthCheck:          healthCheck,
	}

	pool, err := c.loadBalancerService.Pools(loadBalancerID).Create(ctx, create)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create load balancer pool: %s", err))
		return
	}

	err = c.loadBalancerService.WaitUntilMutable(ctx, loadBalancerID)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to wait until load balancer is mutable: %s", err))
		return
	}

	var state computeLoadBalancerPoolResourceData
	state.FromEntity(loadBalancerID, pool)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeLoadBalancerPoolResource) Read(ctx context.Context, request tfsdk.ReadResourceRequest, response *tfsdk.ReadResourceResponse) {
	var state computeLoadBalancerPoolResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	loadBalancerID := int(state.LoadBalancerID.Value)

	pool, err := c.loadBalancerService.Pools(loadBalancerID).Get(ctx, int(state.ID.Value))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get load balancer pool: %s", err))
		return
	}

	state.FromEntity(loadBalancerID, pool)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeLoadBalancerPoolResource) Update(ctx context.Context, request tfsdk.UpdateResourceRequest, response *tfsdk.UpdateResourceResponse) {
	var state computeLoadBalancerPoolResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	var config computeLoadBalancerPoolResourceData
	diagnostics = request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	healthCheck, diagnostics := convertHealthCheckConfigToAPIOptions(*config.HealthCheck)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	loadBalancerID := int(state.LoadBalancerID.Value)
	poolID := int(state.ID.Value)

	update := compute.LoadBalancerPoolUpdate{
		CertificateID:        int(config.CertificateID.Value),
		BalancingAlgorithmID: int(config.BalancingAlgorithmID.Value),
		StickySession:        config.StickySession.Value,
		HealthCheck:          healthCheck,
	}

	pool, err := c.loadBalancerService.Pools(loadBalancerID).Update(ctx, poolID, update)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update load balancer pool: %s", err))
		return
	}

	err = c.loadBalancerService.WaitUntilMutable(ctx, loadBalancerID)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to wait until load balancer is mutable: %s", err))
		return
	}

	state.FromEntity(loadBalancerID, pool)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeLoadBalancerPoolResource) Delete(ctx context.Context, request tfsdk.DeleteResourceRequest, response *tfsdk.DeleteResourceResponse) {
	var state computeLoadBalancerPoolResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	loadBalancerID := int(state.LoadBalancerID.Value)
	poolID := int(state.ID.Value)

	err := c.loadBalancerService.Pools(loadBalancerID).Delete(ctx, poolID)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete load balancer pool: %s", err))
		return
	}

	err = c.loadBalancerService.WaitUntilMutable(ctx, loadBalancerID)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to wait until load balancer is mutable: %s", err))
		return
	}
}

func convertHealthCheckConfigToAPIOptions(config computeLoadBalancerHealthCheckResourceData) (options compute.LoadBalancerHealthCheckOptions, diagnostics diag.Diagnostics) {
	healthCheckIntervalSeconds := 0
	healthCheckTimeoutSeconds := 0

	if !config.Interval.Null {
		duration, err := time.ParseDuration(config.Interval.Value)
		if err != nil {
			diagnostics.AddError("Invalid Interval", fmt.Sprintf("unable to parse health check interval: %s", err))
			return
		}

		healthCheckIntervalSeconds = int(duration.Milliseconds() / 1000)
	}

	if !config.Timeout.Null {
		duration, err := time.ParseDuration(config.Timeout.Value)
		if err != nil {
			diagnostics.AddError("Invalid Timeout", fmt.Sprintf("unable to parse health check timeout: %s", err))
			return
		}

		healthCheckTimeoutSeconds = int(duration.Milliseconds() / 1000)
	}

	options = compute.LoadBalancerHealthCheckOptions{
		TypeID:             int(config.TypeID.Value),
		Interval:           healthCheckIntervalSeconds,
		Timeout:            healthCheckTimeoutSeconds,
		HealthyThreshold:   int(config.HealthyThreshold.Value),
		UnhealthyThreshold: int(config.UnhealthyThreshold.Value),
	}

	if config.HTTP != nil {
		options.HTTPMethod = config.HTTP.Method.Value
		options.HTTPPath = config.HTTP.Path.Value
	}

	return
}
