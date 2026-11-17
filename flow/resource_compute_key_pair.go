package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/v2/compute"
	"github.com/flowswiss/goclient/v2/core"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*computeKeyPairResource)(nil)
	_ resource.ResourceWithConfigure   = (*computeKeyPairResource)(nil)
	_ resource.ResourceWithImportState = (*computeKeyPairResource)(nil)
)

type computeKeyPairResourceData struct {
	ID          types.Int64  `tfsdk:"id"`
	Fingerprint types.String `tfsdk:"fingerprint"`

	Name      types.String `tfsdk:"name"`
	PublicKey types.String `tfsdk:"public_key"`
}

func (d *computeKeyPairResourceData) FromEntity(keyPair compute.KeyPair) {
	d.ID = types.Int64Value(int64(keyPair.ID))
	d.Fingerprint = types.StringValue(keyPair.Fingerprint)
	d.Name = types.StringValue(keyPair.Name)
}

func (c computeKeyPairResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the key pair",
				Computed:            true,
			},
			"fingerprint": schema.StringAttribute{
				MarkdownDescription: "fingerprint of the public key",
				Computed:            true,
			},

			"name": schema.StringAttribute{
				MarkdownDescription: "name of the key pair",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"public_key": schema.StringAttribute{
				MarkdownDescription: "public key of the key pair",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func newComputeKeyPairResource() resource.Resource {
	return &computeKeyPairResource{}
}

func (c *computeKeyPairResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_key_pair"
}

func (c *computeKeyPairResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.client = client
}

type computeKeyPairResource struct {
	client flowClient
}

func (c computeKeyPairResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var config computeKeyPairResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	create := compute.KeyPairCreateReq{
		Name:      config.Name.ValueString(),
		PublicKey: config.PublicKey.ValueString(),
	}

	var keyPair compute.KeyPair
	err := retryCreate(ctx, "create key pair", func() (err error) {
		keyPair, err = c.client.Compute.KeyPair.Create(ctx, create)
		return err
	})
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

func (c computeKeyPairResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state computeKeyPairResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	list, err := c.client.Compute.KeyPair.List(ctx, core.CursorAll)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to list key pairs: %s", err))
		return
	}

	for _, keyPair := range list.Items {
		if keyPair.ID == int(state.ID.ValueInt64()) {
			state.FromEntity(keyPair)

			diagnostics = response.State.Set(ctx, state)
			response.Diagnostics.Append(diagnostics...)
			return
		}
	}

	removeGone(ctx, response, fmt.Sprintf("key pair %d", state.ID.ValueInt64()))
}

func (c computeKeyPairResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	response.Diagnostics.AddError("Not Supported", "updating a key pair is not supported")
}

func (c computeKeyPairResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state computeKeyPairResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	err := retryDelete(ctx, "delete key pair", func() error {
		return c.client.Compute.KeyPair.Delete(ctx, compute.KeyPairDeleteReq{ID: uint(state.ID.ValueInt64())})
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete key pair: %s", err))
		return
	}
}

func (c computeKeyPairResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importStatePassthroughInt64ID(ctx, path.Root("id"), request, response)
}
