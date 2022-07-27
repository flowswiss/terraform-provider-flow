package flow

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/flowswiss/goclient"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ tfsdk.Provider = (*provider)(nil)

var (
	version         = "dev"
	defaultEndpoint = "https://api.flow.swiss/"
)

func New() tfsdk.Provider {
	return &provider{}
}

type provider struct {
	client     goclient.Client
	configured bool
}

type providerData struct {
	Token    types.String `tfsdk:"token"`
	Endpoint types.String `tfsdk:"endpoint"`
}

func (p *provider) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"token": {
				Type:                types.StringType,
				MarkdownDescription: "authentication token for the flow api",
				Optional:            true,
				Sensitive:           true,
			},
			"endpoint": {
				Type:                types.StringType,
				MarkdownDescription: "endpoint for the flow api",
				Optional:            true,
			},
		},
	}, nil
}

func (p *provider) Configure(ctx context.Context, request tfsdk.ConfigureProviderRequest, response *tfsdk.ConfigureProviderResponse) {
	if p.configured {
		return
	}

	var data providerData
	diagnostics := request.Config.Get(ctx, &data)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	if data.Token.Null {
		if val, ok := os.LookupEnv("FLOW_TOKEN"); ok {
			data.Token = types.String{Value: val}
		} else {
			response.Diagnostics.AddError(
				"Missing Token",
				"The token is missing. Please set the token in the provider configuration or set the FLOW_TOKEN environment variable.",
			)
			return
		}
	}

	if data.Endpoint.Null {
		data.Endpoint = types.String{Value: defaultEndpoint}

		if val, ok := os.LookupEnv("FLOW_ENDPOINT"); ok {
			data.Endpoint = types.String{Value: val}
		}
	}

	p.client = goclient.NewClient(
		goclient.WithToken(data.Token.Value),
		goclient.WithBase(data.Endpoint.Value),
		goclient.WithUserAgent(fmt.Sprintf("terraform-provider-flow/%s", version)),

		goclient.WithHTTPClientOption(func(c *http.Client) {
			c.Transport = logTransport{base: c.Transport}
		}),
	)

	p.configured = true
}

func (p *provider) GetResources(ctx context.Context) (map[string]tfsdk.ResourceType, diag.Diagnostics) {
	return map[string]tfsdk.ResourceType{
		"flow_compute_certificate":                  computeCertificateResourceType{},
		"flow_compute_elastic_ip":                   computeElasticIPResourceType{},
		"flow_compute_elastic_ip_server_attachment": computeElasticIPServerAttachmentResourceType{},
		"flow_compute_key_pair":                     computeKeyPairResourceType{},
		"flow_compute_load_balancer_member":         computeLoadBalancerMemberResourceType{},
		"flow_compute_load_balancer_pool":           computeLoadBalancerPoolResourceType{},
		"flow_compute_network":                      computeNetworkResourceType{},
		"flow_compute_network_interface":            computeNetworkInterfaceResourceType{},
		"flow_compute_router":                       computeRouterResourceType{},
		"flow_compute_router_interface":             computeRouterInterfaceResourceType{},
		"flow_compute_router_route":                 computeRouterRouteResourceType{},
		"flow_compute_security_group":               computeSecurityGroupResourceType{},
		"flow_compute_security_group_rule":          computeSecurityGroupRuleResourceType{},
		"flow_compute_snapshot":                     computeSnapshotResourceType{},
		"flow_compute_volume":                       computeVolumeResourceType{},
		"flow_compute_volume_attachment":            computeVolumeAttachmentResourceType{},

		"flow_mac_bare_metal_elastic_ip": macBareMetalElasticIPResourceType{},
	}, nil
}

func (p *provider) GetDataSources(ctx context.Context) (map[string]tfsdk.DataSourceType, diag.Diagnostics) {
	return map[string]tfsdk.DataSourceType{
		"flow_module":   moduleDataSourceType{},
		"flow_location": locationDataSourceType{},

		"flow_compute_certificate":                     computeCertificateDataSourceType{},
		"flow_compute_elastic_ip":                      computeElasticIPDataSourceType{},
		"flow_compute_key_pair":                        computeKeyPairDataSourceType{},
		"flow_compute_load_balancer_algorithm":         computeLoadBalancerAlgorithmDataSourceType{},
		"flow_compute_load_balancer_health_check_type": computeLoadBalancerHealthCheckTypeDataSourceType{},
		"flow_compute_load_balancer_member":            computeLoadBalancerMemberDataSourceType{},
		"flow_compute_load_balancer_pool":              computeLoadBalancerPoolDataSourceType{},
		"flow_compute_load_balancer_protocol":          computeLoadBalancerProtocolDataSourceType{},
		"flow_compute_network":                         computeNetworkDataSourceType{},
		"flow_compute_network_interface":               computeNetworkInterfaceDataSourceType{},
		"flow_compute_router":                          computeRouterDataSourceType{},
		"flow_compute_router_interface":                computeRouterInterfaceDataSourceType{},
		"flow_compute_router_route":                    computeRouterRouteDataSourceType{},
		"flow_compute_security_group":                  computeSecurityGroupDataSourceType{},
		"flow_compute_security_group_rule":             computeSecurityGroupRuleDataSourceType{},
		"flow_compute_snapshot":                        computeSnapshotDataSourceType{},
		"flow_compute_volume":                          computeVolumeDataSourceType{},
	}, nil
}

func convertToLocalProviderType(p tfsdk.Provider) (prov *provider, diagnostics diag.Diagnostics) {
	prov, ok := p.(*provider)
	if !ok {
		diagnostics.AddError(
			"Unexpected Provider Instance Type",
			fmt.Sprintf("While creating the data source or resource, an unexpected provider type (%T) was received. This is always a bug in the provider code and should be reported to the provider developers.", p),
		)

		return
	}

	return
}

func waitForCondition(ctx context.Context, check func(ctx context.Context) (bool, diag.Diagnostics)) (diagnostics diag.Diagnostics) {
	done, d := check(ctx)
	diagnostics.Append(d...)
	if done || diagnostics.HasError() {
		return
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:

		case <-ctx.Done():
			diagnostics.AddError("Timeout", "Timeout while waiting for condition")
			return
		}

		done, d = check(ctx)
		diagnostics.Append(d...)
		if done || diagnostics.HasError() {
			return
		}
	}
}

type logTransport struct {
	base http.RoundTripper
}

func (l logTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	additionalContext := map[string]interface{}{
		"method": req.Method,
		"url":    req.URL.String(),
	}

	res, err := l.transport().RoundTrip(req)

	if err == nil {
		additionalContext["request_id"] = res.Header.Get("X-Request-ID")

		msg := fmt.Sprintf("request to `%s %s` resulted in `%s`", req.Method, req.URL.String(), res.Status)
		tflog.Trace(req.Context(), msg, additionalContext)
	} else {
		msg := fmt.Sprintf("request to `%s %s` resulted in `%s`", req.Method, req.URL.String(), err)
		tflog.Trace(req.Context(), msg, additionalContext)
	}

	return res, err
}

func (l logTransport) transport() http.RoundTripper {
	if l.base == nil {
		return http.DefaultTransport
	}

	return l.base
}
