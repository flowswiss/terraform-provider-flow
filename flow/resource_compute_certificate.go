package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/compute"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var (
	_ tfsdk.ResourceType            = (*computeCertificateResourceType)(nil)
	_ tfsdk.Resource                = (*computeCertificateResource)(nil)
	_ tfsdk.ResourceWithImportState = (*computeCertificateResource)(nil)
)

type computeCertificateResourceAttributes struct {
	CommonName         types.String `tfsdk:"common_name"`
	OrganizationalUnit types.String `tfsdk:"organizational_unit"`
	Organization       types.String `tfsdk:"organization"`
	Locality           types.String `tfsdk:"locality"`
	Province           types.String `tfsdk:"province"`
	Country            types.String `tfsdk:"country"`
}

type computeCertificateResourceInfo struct {
	Subject *computeCertificateResourceAttributes `tfsdk:"subject"`
	Issuer  *computeCertificateResourceAttributes `tfsdk:"issuer"`

	NotBefore types.String `tfsdk:"not_before"`
	NotAfter  types.String `tfsdk:"not_after"`

	SerialNumber types.String `tfsdk:"serial_number"`
}

type computeCertificateResourceData struct {
	ID         types.Int64  `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	LocationID types.Int64  `tfsdk:"location_id"`

	Certificate types.String `tfsdk:"certificate"`
	PrivateKey  types.String `tfsdk:"private_key"`

	Info *computeCertificateResourceInfo `tfsdk:"info"`
}

func (c *computeCertificateResourceData) FromEntity(certificate compute.Certificate) {
	c.ID = types.Int64{Value: int64(certificate.ID)}
	c.Name = types.String{Value: certificate.Name}
	c.LocationID = types.Int64{Value: int64(certificate.Location.ID)}

	c.Info = &computeCertificateResourceInfo{
		Subject: &computeCertificateResourceAttributes{
			CommonName:         types.String{Value: certificate.Details.Subject["CN"]},
			OrganizationalUnit: types.String{Value: certificate.Details.Subject["OU"]},
			Organization:       types.String{Value: certificate.Details.Subject["O"]},
			Locality:           types.String{Value: certificate.Details.Subject["L"]},
			Province:           types.String{Value: certificate.Details.Subject["P"]},
			Country:            types.String{Value: certificate.Details.Subject["C"]},
		},
		Issuer: &computeCertificateResourceAttributes{
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

type computeCertificateResourceType struct{}

func (c computeCertificateResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
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

func (c computeCertificateResourceType) NewResource(ctx context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return computeCertificateResource{
		certificateService: compute.NewCertificateService(prov.client),
	}, diagnostics
}

type computeCertificateResource struct {
	certificateService compute.CertificateService
}

func (c computeCertificateResource) Create(ctx context.Context, request tfsdk.CreateResourceRequest, response *tfsdk.CreateResourceResponse) {
	var config computeCertificateResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	create := compute.CertificateCreate{
		Name:        config.Name.Value,
		LocationID:  int(config.LocationID.Value),
		Certificate: config.Certificate.Value,
		PrivateKey:  config.PrivateKey.Value,
	}

	certificate, err := c.certificateService.Create(ctx, create)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create certificate: %s", err))
		return
	}

	var state computeCertificateResourceData
	state.FromEntity(certificate)

	// copy the certificate and private key from the config because the api does not return it
	state.Certificate = config.Certificate
	state.PrivateKey = config.PrivateKey

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeCertificateResource) Read(ctx context.Context, request tfsdk.ReadResourceRequest, response *tfsdk.ReadResourceResponse) {
	var state computeCertificateResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := c.certificateService.List(ctx, goclient.Cursor{NoFilter: 1})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list certificates: %s", err))
		return
	}

	for _, certificate := range list.Items {
		if certificate.ID == int(state.ID.Value) {
			state.FromEntity(certificate)

			diagnostics = response.State.Set(ctx, state)
			response.Diagnostics.Append(diagnostics...)
			return
		}
	}

	response.Diagnostics.AddError("Not Found", fmt.Sprintf("certificate with id %d not found", state.ID.Value))
}

func (c computeCertificateResource) Update(ctx context.Context, request tfsdk.UpdateResourceRequest, response *tfsdk.UpdateResourceResponse) {
	response.Diagnostics.AddError("Not Supported", "updating a certificate is not supported")
}

func (c computeCertificateResource) Delete(ctx context.Context, request tfsdk.DeleteResourceRequest, response *tfsdk.DeleteResourceResponse) {
	var state computeCertificateResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	err := c.certificateService.Delete(ctx, int(state.ID.Value))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete certificate: %s", err))
		return
	}
}

func (c computeCertificateResource) ImportState(ctx context.Context, request tfsdk.ImportResourceStateRequest, response *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, tftypes.NewAttributePath().WithAttributeName("id"), request, response)
}
