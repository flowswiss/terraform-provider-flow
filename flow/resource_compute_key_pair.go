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
	_ tfsdk.ResourceType            = (*computeKeyPairResourceType)(nil)
	_ tfsdk.Resource                = (*computeKeyPairResource)(nil)
	_ tfsdk.ResourceWithImportState = (*computeKeyPairResource)(nil)
)

type computeKeyPairResourceData struct {
	ID          types.Int64  `tfsdk:"id"`
	Fingerprint types.String `tfsdk:"fingerprint"`

	Name      types.String `tfsdk:"name"`
	PublicKey types.String `tfsdk:"public_key"`
}

func (d *computeKeyPairResourceData) FromEntity(keyPair compute.KeyPair) {
	d.ID = types.Int64{Value: int64(keyPair.ID)}
	d.Fingerprint = types.String{Value: keyPair.Fingerprint}
	d.Name = types.String{Value: keyPair.Name}
}

type computeKeyPairResourceType struct{}

func (c computeKeyPairResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the key pair",
				Computed:            true,
			},
			"fingerprint": {
				Type:                types.StringType,
				MarkdownDescription: "fingerprint of the public key",
				Computed:            true,
			},

			"name": {
				Type:                types.StringType,
				MarkdownDescription: "name of the key pair",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"public_key": {
				Type:                types.StringType,
				MarkdownDescription: "public key of the key pair",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
		},
	}, nil
}

func (c computeKeyPairResourceType) NewResource(ctx context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return computeKeyPairResource{
		keyPairService: compute.NewKeyPairService(prov.client),
	}, diagnostics
}

type computeKeyPairResource struct {
	keyPairService compute.KeyPairService
}

func (c computeKeyPairResource) Create(ctx context.Context, request tfsdk.CreateResourceRequest, response *tfsdk.CreateResourceResponse) {
	var config computeKeyPairResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	create := compute.KeyPairCreate{
		Name:      config.Name.Value,
		PublicKey: config.PublicKey.Value,
	}

	keyPair, err := c.keyPairService.Create(ctx, create)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create key pair: %s", err))
		return
	}

	var state computeKeyPairResourceData
	state.FromEntity(keyPair)

	// copy the public key from the config because the api does not return it
	state.PublicKey = config.PublicKey

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeKeyPairResource) Read(ctx context.Context, request tfsdk.ReadResourceRequest, response *tfsdk.ReadResourceResponse) {
	var state computeKeyPairResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := c.keyPairService.List(ctx, goclient.Cursor{NoFilter: 1})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list key pairs: %s", err))
		return
	}

	for _, keyPair := range list.Items {
		if keyPair.ID == int(state.ID.Value) {
			state.FromEntity(keyPair)

			diagnostics = response.State.Set(ctx, state)
			response.Diagnostics.Append(diagnostics...)
			return
		}
	}

	response.Diagnostics.AddError("Not Found", fmt.Sprintf("key pair with id %d not found", state.ID.Value))
}

func (c computeKeyPairResource) Update(ctx context.Context, request tfsdk.UpdateResourceRequest, response *tfsdk.UpdateResourceResponse) {
	response.Diagnostics.AddError("Not Supported", "updating a key pair is not supported")
}

func (c computeKeyPairResource) Delete(ctx context.Context, request tfsdk.DeleteResourceRequest, response *tfsdk.DeleteResourceResponse) {
	var state computeKeyPairResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	err := c.keyPairService.Delete(ctx, int(state.ID.Value))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete key pair: %s", err))
		return
	}
}

func (c computeKeyPairResource) ImportState(ctx context.Context, request tfsdk.ImportResourceStateRequest, response *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, tftypes.NewAttributePath().WithAttributeName("id"), request, response)
}
