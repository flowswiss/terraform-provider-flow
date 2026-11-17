package flow

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/flowswiss/goclient"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ tfsdk.Provider = (*provider)(nil)

type Option func(p *provider)

func WithVersion(version string) Option {
	return func(p *provider) {
		p.version = version
	}
}

func WithDefaultEndpoint(endpoint string) Option {
	return func(p *provider) {
		p.defaultEndpoint = endpoint
	}
}

func New(opts ...Option) tfsdk.Provider {
	p := &provider{
		version:         "dev",
		defaultEndpoint: "https://api.flow.swiss/",
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

type provider struct {
	version         string
	defaultEndpoint string

	client     goclient.Client
	configured bool
}

type providerData struct {
	Token        types.String `tfsdk:"token"`
	Endpoint     types.String `tfsdk:"endpoint"`
	RetryTimeout types.String `tfsdk:"retry_timeout"`
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
			"retry_timeout": {
				Type:                types.StringType,
				MarkdownDescription: "how long a failing api call is retried before the error is reported, as a duration such as `90s` or `2m` (default `90s`, `0` disables retries). can also be set with the `FLOW_RETRY_TIMEOUT` environment variable",
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
		data.Endpoint = types.String{Value: p.defaultEndpoint}

		if val, ok := os.LookupEnv("FLOW_ENDPOINT"); ok {
			data.Endpoint = types.String{Value: val}
		}
	}

	if data.RetryTimeout.Null {
		if val, ok := os.LookupEnv("FLOW_RETRY_TIMEOUT"); ok {
			data.RetryTimeout = types.String{Value: val}
		}
	}

	if !data.RetryTimeout.Null {
		timeout, err := parseRetryTimeout(data.RetryTimeout.Value)
		if err != nil {
			response.Diagnostics.AddAttributeError(
				path.Root("retry_timeout"),
				"Invalid Retry Timeout",
				err.Error(),
			)
			return
		}

		defaultRetryPolicy.Timeout = timeout
	}

	p.client = goclient.NewClient(
		goclient.WithToken(data.Token.Value),
		goclient.WithBase(data.Endpoint.Value),
		goclient.WithUserAgent(fmt.Sprintf("terraform-provider-flow/%s", p.version)),

		goclient.WithHTTPClientOption(installTransport),
	)

	p.configured = true
}

func (p *provider) GetResources(ctx context.Context) (map[string]tfsdk.ResourceType, diag.Diagnostics) {
	return map[string]tfsdk.ResourceType{
		"flow_compute_certificate":                  computeCertificateResourceType{},
		"flow_compute_elastic_ip":                   computeElasticIPResourceType{},
		"flow_compute_elastic_ip_server_attachment": computeElasticIPServerAttachmentResourceType{},
		"flow_compute_key_pair":                     computeKeyPairResourceType{},
		"flow_compute_load_balancer":                computeLoadBalancerResourceType{},
		"flow_compute_load_balancer_member":         computeLoadBalancerMemberResourceType{},
		"flow_compute_load_balancer_pool":           computeLoadBalancerPoolResourceType{},
		"flow_compute_network":                      computeNetworkResourceType{},
		"flow_compute_network_interface":            computeNetworkInterfaceResourceType{},
		"flow_compute_router":                       computeRouterResourceType{},
		"flow_compute_router_interface":             computeRouterInterfaceResourceType{},
		"flow_compute_router_route":                 computeRouterRouteResourceType{},
		"flow_compute_security_group":               computeSecurityGroupResourceType{},
		"flow_compute_security_group_rule":          computeSecurityGroupRuleResourceType{},
		"flow_compute_server":                       computeServerResourceType{},
		"flow_compute_volume":                       computeVolumeResourceType{},
		"flow_compute_volume_attachment":            computeVolumeAttachmentResourceType{},

		"flow_kubernetes_cluster": kubernetesClusterResourceType{},

		"flow_mac_bare_metal_device":                macBareMetalDeviceResourceType{},
		"flow_mac_bare_metal_elastic_ip":            macBareMetalElasticIPResourceType{},
		"flow_mac_bare_metal_elastic_ip_attachment": macBareMetalElasticIPDeviceAttachmentResourceType{},
		"flow_mac_bare_metal_network":               macBareMetalNetworkResourceType{},
		"flow_mac_bare_metal_security_group":        macBareMetalSecurityGroupResourceType{},
		"flow_mac_bare_metal_security_group_rule":   macBareMetalSecurityGroupRuleResourceType{},
	}, nil
}

func (p *provider) GetDataSources(ctx context.Context) (map[string]tfsdk.DataSourceType, diag.Diagnostics) {
	return map[string]tfsdk.DataSourceType{
		"flow_location": locationDataSourceType{},
		"flow_module":   moduleDataSourceType{},
		"flow_product":  productDataSourceType{},

		"flow_compute_certificate":                     computeCertificateDataSourceType{},
		"flow_compute_elastic_ip":                      computeElasticIPDataSourceType{},
		"flow_compute_image":                           computeImageDataSourceType{},
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
		"flow_compute_server":                          computeServerDataSourceType{},
		"flow_compute_snapshot":                        computeSnapshotDataSourceType{},
		"flow_compute_volume":                          computeVolumeDataSourceType{},

		"flow_kubernetes_cluster":     kubernetesClusterDataSourceType{},
		"flow_kubernetes_kube_config": kubernetesKubeConfigDataSourceType{},

		"flow_mac_bare_metal_elastic_ip":          macBareMetalElasticIPDataSourceType{},
		"flow_mac_bare_metal_network":             macBareMetalNetworkDataSourceType{},
		"flow_mac_bare_metal_security_group":      macBareMetalSecurityGroupDataSourceType{},
		"flow_mac_bare_metal_security_group_rule": macBareMetalSecurityGroupRuleDataSourceType{},
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

func installTransport(c *http.Client) {
	base := http.DefaultTransport.(*http.Transport).Clone()
	base.ResponseHeaderTimeout = responseHeaderTimeout

	// goclient's WithToken put its auth transport in front of the default one —
	// replacing it drops the authorization header
	var inner http.RoundTripper = base
	switch t := c.Transport.(type) {
	case goclient.AuthTransport:
		t.Base = base
		inner = t
	case nil:
	default:
		inner = t
	}

	// the read retry sits outside the log transport so every attempt is traced
	c.Transport = readRetryTransport{base: logTransport{base: inner}}
}
