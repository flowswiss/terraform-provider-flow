package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/compute"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/flowswiss/terraform-provider-flow/filter"
)

var (
	_ tfsdk.DataSourceType = (*computeCertificateDataSourceType)(nil)
	_ tfsdk.DataSource     = (*computeCertificateDataSource)(nil)
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
	c.ID = types.Int64{Value: int64(certificate.ID)}
	c.Name = types.String{Value: certificate.Name}
	c.LocationID = types.Int64{Value: int64(certificate.Location.ID)}

	c.Info = &computeCertificateDataSourceInfo{
		Subject: &computeCertificateDataSourceAttributes{
			CommonName:         types.String{Value: certificate.Details.Subject["CN"]},
			OrganizationalUnit: types.String{Value: certificate.Details.Subject["OU"]},
			Organization:       types.String{Value: certificate.Details.Subject["O"]},
			Locality:           types.String{Value: certificate.Details.Subject["L"]},
			Province:           types.String{Value: certificate.Details.Subject["P"]},
			Country:            types.String{Value: certificate.Details.Subject["C"]},
		},
		Issuer: &computeCertificateDataSourceAttributes{
			CommonName:         types.String{Value: certificate.Details.Issuer["CN"]},
			OrganizationalUnit: types.String{Value: certificate.Details.Issuer["OU"]},
			Organization:       types.String{Value: certificate.Details.Issuer["O"]},
			Locality:           types.String{Value: certificate.Details.Issuer["L"]},
			Province:           types.String{Value: certificate.Details.Issuer["P"]},
			Country:            types.String{Value: certificate.Details.Issuer["C"]},
		},
		NotBefore:    types.String{Value: certificate.Details.ValidFrom.String()},
		NotAfter:     types.String{Value: certificate.Details.ValidTo.String()},
		SerialNumber: types.String{Value: certificate.Details.Serial},
	}
}

func (c computeCertificateDataSourceData) AppliesTo(certificate compute.Certificate) bool {
	if !c.ID.Null && c.ID.Value != int64(certificate.ID) {
		return false
	}

	if !c.Name.Null && c.Name.Value != certificate.Name {
		return false
	}

	if !c.LocationID.Null && c.LocationID.Value != int64(certificate.Location.ID) {
		return false
	}

	return true
}

type computeCertificateDataSourceType struct{}

func (c computeCertificateDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	certificateInfoAttributes := map[string]tfsdk.Attribute{
		"common_name": {
			Type:                types.StringType,
			MarkdownDescription: "common name of the certificate (CN)",
			Computed:            true,
		},
		"organizational_unit": {
			Type:                types.StringType,
			MarkdownDescription: "organizational unit of the certificate (OU)",
			Computed:            true,
		},
		"organization": {
			Type:                types.StringType,
			MarkdownDescription: "organization of the certificate (O)",
			Computed:            true,
		},
		"locality": {
			Type:                types.StringType,
			MarkdownDescription: "locality of the certificate (L)",
			Computed:            true,
		},
		"province": {
			Type:                types.StringType,
			MarkdownDescription: "province of the certificate (S)",
			Computed:            true,
		},
		"country": {
			Type:                types.StringType,
			MarkdownDescription: "country of the certificate (C)",
			Computed:            true,
		},
	}

	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the certificate",
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},
			"name": {
				Type:                types.StringType,
				MarkdownDescription: "name of the certificate",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"location_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the location",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"certificate": {
				Type:                types.StringType,
				MarkdownDescription: "certificate in base64 encoded PEM format",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"private_key": {
				Type:                types.StringType,
				MarkdownDescription: "private key in base64 encoded PEM format",
				Required:            true,
				Sensitive:           true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"info": {
				Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
					"subject": {
						Attributes:          tfsdk.SingleNestedAttributes(certificateInfoAttributes),
						MarkdownDescription: "subject of the certificate",
						Computed:            true,
					},
					"issuer": {
						Attributes:          tfsdk.SingleNestedAttributes(certificateInfoAttributes),
						MarkdownDescription: "issuer of the certificate",
						Computed:            true,
					},
					"not_before": {
						Type:                types.StringType,
						MarkdownDescription: "not before date of the certificate",
						Computed:            true,
					},
					"not_after": {
						Type:                types.StringType,
						MarkdownDescription: "not after date of the certificate",
						Computed:            true,
					},
					"serial_number": {
						Type:                types.StringType,
						MarkdownDescription: "serial number of the certificate",
						Computed:            true,
					},
				}),
				MarkdownDescription: "information about the certificate",
				Computed:            true,
			},
		},
	}, nil
}

func (c computeCertificateDataSourceType) NewDataSource(ctx context.Context, p tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return computeCertificateDataSource{
		certificateService: compute.NewCertificateService(prov.client),
	}, diagnostics
}

type computeCertificateDataSource struct {
	certificateService compute.CertificateService
}

func (c computeCertificateDataSource) Read(ctx context.Context, request tfsdk.ReadDataSourceRequest, response *tfsdk.ReadDataSourceResponse) {
	var config computeCertificateDataSourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := c.certificateService.List(ctx, goclient.Cursor{NoFilter: 1})
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
