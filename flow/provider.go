package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
				MarkdownDescription: "Authentication token for the Flow API",
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
	)

	p.configured = true
}

func (p *provider) GetResources(ctx context.Context) (map[string]tfsdk.ResourceType, diag.Diagnostics) {
	return map[string]tfsdk.ResourceType{
		"flow_volume": computeVolumeResourceType{},
	}, nil
}

func (p *provider) GetDataSources(ctx context.Context) (map[string]tfsdk.DataSourceType, diag.Diagnostics) {
	return map[string]tfsdk.DataSourceType{
		"flow_module":   moduleDataSourceType{},
		"flow_location": locationDataSourceType{},
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
