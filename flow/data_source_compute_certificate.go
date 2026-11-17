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
	_ datasource.DataSource              = (*computeCertificateDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*computeCertificateDataSource)(nil)
)

type computeCertificateDataSourceAttributes struct {
	CommonName         types.String `tfsdk:"common_name"`
	OrganizationalUnit types.String `tfsdk:"organizational_unit"`
	Organization       types.String `tfsdk:"organization"`
	Locality           types.String `tfsdk:"locality"`
	Province           types.String `tfsdk:"province"`
	Country            types.String `tfsdk:"country"`
}

type computeCertificateDataSourceInfo struct {
	Subject *computeCertificateDataSourceAttributes `tfsdk:"subject"`
	Issuer  *computeCertificateDataSourceAttributes `tfsdk:"issuer"`

	NotBefore types.String `tfsdk:"not_before"`
	NotAfter  types.String `tfsdk:"not_after"`

	SerialNumber types.String `tfsdk:"serial_number"`
}

type computeCertificateDataSourceData struct {
	ID         types.Int64  `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	LocationID types.Int64  `tfsdk:"location_id"`

	Info *computeCertificateDataSourceInfo `tfsdk:"info"`
}

func (c *computeCertificateDataSourceData) FromEntity(certificate compute.Certificate) {
	c.ID = types.Int64Value(int64(certificate.ID))
	c.Name = types.StringValue(certificate.Name)
	c.LocationID = types.Int64Value(int64(certificate.Location.ID))

	c.Info = &computeCertificateDataSourceInfo{
		Subject: &computeCertificateDataSourceAttributes{
			CommonName:         types.StringValue(certificate.Details.Subject["CN"]),
			OrganizationalUnit: types.StringValue(certificate.Details.Subject["OU"]),
			Organization:       types.StringValue(certificate.Details.Subject["O"]),
			Locality:           types.StringValue(certificate.Details.Subject["L"]),
			Province:           types.StringValue(certificate.Details.Subject["P"]),
			Country:            types.StringValue(certificate.Details.Subject["C"]),
		},
		Issuer: &computeCertificateDataSourceAttributes{
			CommonName:         types.StringValue(certificate.Details.Issuer["CN"]),
			OrganizationalUnit: types.StringValue(certificate.Details.Issuer["OU"]),
			Organization:       types.StringValue(certificate.Details.Issuer["O"]),
			Locality:           types.StringValue(certificate.Details.Issuer["L"]),
			Province:           types.StringValue(certificate.Details.Issuer["P"]),
			Country:            types.StringValue(certificate.Details.Issuer["C"]),
		},
		NotBefore:    types.StringValue(certificate.Details.ValidFrom.String()),
		NotAfter:     types.StringValue(certificate.Details.ValidTo.String()),
		SerialNumber: types.StringValue(certificate.Details.Serial),
	}
}

func (c computeCertificateDataSourceData) AppliesTo(certificate compute.Certificate) bool {
	if !c.ID.IsNull() && c.ID.ValueInt64() != int64(certificate.ID) {
		return false
	}

	if !c.Name.IsNull() && c.Name.ValueString() != certificate.Name {
		return false
	}

	if !c.LocationID.IsNull() && c.LocationID.ValueInt64() != int64(certificate.Location.ID) {
		return false
	}

	return true
}

func (c computeCertificateDataSource) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	certificateInfoAttributes := map[string]schema.Attribute{
		"common_name": schema.StringAttribute{
			MarkdownDescription: "common name of the certificate (CN)",
			Computed:            true,
		},
		"organizational_unit": schema.StringAttribute{
			MarkdownDescription: "organizational unit of the certificate (OU)",
			Computed:            true,
		},
		"organization": schema.StringAttribute{
			MarkdownDescription: "organization of the certificate (O)",
			Computed:            true,
		},
		"locality": schema.StringAttribute{
			MarkdownDescription: "locality of the certificate (L)",
			Computed:            true,
		},
		"province": schema.StringAttribute{
			MarkdownDescription: "province of the certificate (S)",
			Computed:            true,
		},
		"country": schema.StringAttribute{
			MarkdownDescription: "country of the certificate (C)",
			Computed:            true,
		},
	}

	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the certificate",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the certificate",
				Required:            true,
			},
			"location_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the location",
				Required:            true,
			},
			"info": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"subject": schema.SingleNestedAttribute{
						Attributes:          certificateInfoAttributes,
						MarkdownDescription: "subject of the certificate",
						Computed:            true,
					},
					"issuer": schema.SingleNestedAttribute{
						Attributes:          certificateInfoAttributes,
						MarkdownDescription: "issuer of the certificate",
						Computed:            true,
					},
					"not_before": schema.StringAttribute{
						MarkdownDescription: "not before date of the certificate",
						Computed:            true,
					},
					"not_after": schema.StringAttribute{
						MarkdownDescription: "not after date of the certificate",
						Computed:            true,
					},
					"serial_number": schema.StringAttribute{
						MarkdownDescription: "serial number of the certificate",
						Computed:            true,
					},
				},
				MarkdownDescription: "information about the certificate",
				Computed:            true,
			},
		},
	}
}

func newComputeCertificateDataSource() datasource.DataSource {
	return &computeCertificateDataSource{}
}

func (c *computeCertificateDataSource) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_certificate"
}

func (c *computeCertificateDataSource) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.client = client
}

type computeCertificateDataSource struct {
	client flowClient
}

func (c computeCertificateDataSource) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var config computeCertificateDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := c.client.Compute.Certificate.List(ctx, core.CursorAll)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list certificates: %s", err))
		return
	}

	certificate, err := filter.FindOne(config, list.Items)
	if err != nil {
		response.Diagnostics.AddError("Not Found", fmt.Sprintf("unable to find certificate: %s", err))
		return
	}

	var state computeCertificateDataSourceData
	state.FromEntity(certificate)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}
