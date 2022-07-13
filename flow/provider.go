package flow

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/flowswiss/goclient"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ tfsdk.Provider = (*provider)(nil)

func New(version string) func() tfsdk.Provider {
	return func() tfsdk.Provider {
		return &provider{version: version}
	}
}

type provider struct {
	client     goclient.Client
	configured bool

	version string
}

type providerData struct {
	Token string `tfsdk:"token"`
}

func (p *provider) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"token": {
				Type:                types.StringType,
				MarkdownDescription: "authentication token for the flow api",
				Required:            true,
				Sensitive:           true,
			},
		},
	}, nil
}

func (p *provider) Configure(ctx context.Context, request tfsdk.ConfigureProviderRequest, response *tfsdk.ConfigureProviderResponse) {
	var data providerData

	diagnostics := request.Config.Get(ctx, &data)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	if p.configured {
		return
	}

	p.client = goclient.NewClient(
		goclient.WithToken(data.Token),
		goclient.WithUserAgent(fmt.Sprintf("terraform-provider-flow/%s", p.version)),

		goclient.WithHTTPClientOption(func(c *http.Client) {
			c.Transport = logTransport{base: c.Transport}
		}),
	)

	p.configured = true
}

func (p *provider) GetResources(ctx context.Context) (map[string]tfsdk.ResourceType, diag.Diagnostics) {
	return map[string]tfsdk.ResourceType{
		"flow_compute_key_pair":          computeKeyPairResourceType{},
		"flow_compute_network":           computeNetworkResourceType{},
		"flow_compute_router":            computeRouterResourceType{},
		"flow_compute_router_interface":  computeRouterInterfaceResourceType{},
		"flow_compute_snapshot":          computeSnapshotResourceType{},
		"flow_compute_volume":            computeVolumeResourceType{},
		"flow_compute_volume_attachment": computeVolumeAttachmentResourceType{},
	}, nil
}

func (p *provider) GetDataSources(ctx context.Context) (map[string]tfsdk.DataSourceType, diag.Diagnostics) {
	return map[string]tfsdk.DataSourceType{
		"flow_module":   moduleDataSourceType{},
		"flow_location": locationDataSourceType{},

		"flow_compute_key_pair":         computeKeyPairDataSourceType{},
		"flow_compute_network":          computeNetworkDataSourceType{},
		"flow_compute_router":           computeRouterDataSourceType{},
		"flow_compute_router_interface": computeRouterInterfaceDataSourceType{},
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

		done, d := check(ctx)
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
