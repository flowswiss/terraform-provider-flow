package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/v2/compute"
	"github.com/flowswiss/goclient/v2/core"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*computeCertificateResource)(nil)
	_ resource.ResourceWithConfigure   = (*computeCertificateResource)(nil)
	_ resource.ResourceWithImportState = (*computeCertificateResource)(nil)
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
	c.ID = types.Int64Value(int64(certificate.ID))
	c.Name = types.StringValue(certificate.Name)
	c.LocationID = types.Int64Value(int64(certificate.Location.ID))

	c.Info = &computeCertificateResourceInfo{
		Subject: &computeCertificateResourceAttributes{
			CommonName:         types.StringValue(certificate.Details.Subject["CN"]),
			OrganizationalUnit: types.StringValue(certificate.Details.Subject["OU"]),
			Organization:       types.StringValue(certificate.Details.Subject["O"]),
			Locality:           types.StringValue(certificate.Details.Subject["L"]),
			Province:           types.StringValue(certificate.Details.Subject["P"]),
			Country:            types.StringValue(certificate.Details.Subject["C"]),
		},
		Issuer: &computeCertificateResourceAttributes{
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

func (c computeCertificateResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
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
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the certificate",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"location_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the location",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"certificate": schema.StringAttribute{
				MarkdownDescription: "certificate in base64 encoded PEM format; editing it produces no plan, rotate with `terraform apply -replace=`",
				Required:            true,
				WriteOnly:           true,
			},
			"private_key": schema.StringAttribute{
				MarkdownDescription: "private key in base64 encoded PEM format; editing it produces no plan, rotate with `terraform apply -replace=`",
				Required:            true,
				Sensitive:           true,
				WriteOnly:           true,
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

func newComputeCertificateResource() resource.Resource {
	return &computeCertificateResource{}
}

func (c *computeCertificateResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_certificate"
}

func (c *computeCertificateResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.client = client
}

type computeCertificateResource struct {
	client flowClient
}

func (c computeCertificateResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var config computeCertificateResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	create := compute.CertificateCreateReq{
		Name:        config.Name.ValueString(),
		LocationID:  int(config.LocationID.ValueInt64()),
		Certificate: config.Certificate.ValueString(),
		PrivateKey:  config.PrivateKey.ValueString(),
	}

	var certificate compute.Certificate
	err := retryCreate(ctx, "create certificate", func() (err error) {
		certificate, err = c.client.Compute.Certificate.Create(ctx, create)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create certificate: %s", err))
		return
	}

	var state computeCertificateResourceData
	state.FromEntity(certificate)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeCertificateResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state computeCertificateResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := c.client.Compute.Certificate.List(ctx, core.CursorAll)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list certificates: %s", err))
		return
	}

	for _, certificate := range list.Items {
		if certificate.ID == int(state.ID.ValueInt64()) {
			state.FromEntity(certificate)

			diagnostics = response.State.Set(ctx, state)
			response.Diagnostics.Append(diagnostics...)
			return
		}
	}

	removeGone(ctx, response, fmt.Sprintf("certificate %d", state.ID.ValueInt64()))
}

func (c computeCertificateResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	response.Diagnostics.AddError("Not Supported", "updating a certificate is not supported")
}

func (c computeCertificateResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state computeCertificateResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	err := retryDelete(ctx, "delete certificate", func() error {
		return c.client.Compute.Certificate.Delete(ctx, compute.CertificateDeleteReq{ID: uint(state.ID.ValueInt64())})
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete certificate: %s", err))
		return
	}
}

func (c computeCertificateResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importStatePassthroughInt64ID(ctx, path.Root("id"), request, response)
}
