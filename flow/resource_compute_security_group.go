package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/v2/compute"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*computeSecurityGroupResource)(nil)
	_ resource.ResourceWithConfigure   = (*computeSecurityGroupResource)(nil)
	_ resource.ResourceWithImportState = (*computeSecurityGroupResource)(nil)
)

type computeSecurityGroupResourceData struct {
	ID         types.Int64  `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	LocationID types.Int64  `tfsdk:"location_id"`
}

func (c *computeSecurityGroupResourceData) FromEntity(securityGroup compute.SecurityGroup) {
	c.ID = types.Int64Value(int64(securityGroup.ID))
	c.Name = types.StringValue(securityGroup.Name)
	c.LocationID = types.Int64Value(int64(securityGroup.Location.ID))
}

func (c computeSecurityGroupResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the security group",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the security group",
				Required:            true,
			},
			"location_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the location",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func newComputeSecurityGroupResource() resource.Resource {
	return &computeSecurityGroupResource{}
}

func (c *computeSecurityGroupResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_security_group"
}

func (c *computeSecurityGroupResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.client = client
}

type computeSecurityGroupResource struct {
	client flowClient
}

func (c computeSecurityGroupResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var config computeSecurityGroupResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	create := compute.SecurityGroupCreateReq{
		Name:       config.Name.ValueString(),
		LocationID: int(config.LocationID.ValueInt64()),
	}

	var securityGroup compute.SecurityGroup
	err := retryCreate(ctx, "create security group", func() (err error) {
		securityGroup, err = c.client.Compute.SecurityGroup.Create(ctx, create)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create security group: %s", err))
		return
	}

	var state computeSecurityGroupResourceData
	state.FromEntity(securityGroup)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeSecurityGroupResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state computeSecurityGroupResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	securityGroup, err := c.client.Compute.SecurityGroup.Get(ctx, compute.SecurityGroupGetReq{ID: uint(state.ID.ValueInt64())})
	if err != nil {
		if isNotFound(err) {
			removeGone(ctx, response, fmt.Sprintf("security group %d", state.ID.ValueInt64()))
			return
		}
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get security group: %s", err))
		return
	}

	state.FromEntity(securityGroup)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeSecurityGroupResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var state computeSecurityGroupResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	var config computeSecurityGroupResourceData
	diagnostics = request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	current, err := c.client.Compute.SecurityGroup.Get(ctx, compute.SecurityGroupGetReq{ID: uint(state.ID.ValueInt64())})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get security group: %s", err))
		return
	}

	update := compute.SecurityGroupUpdateReq{
		ID:          uint(state.ID.ValueInt64()),
		Name:        new(config.Name.ValueString()),
		Description: new(current.Description),
	}

	var securityGroup compute.SecurityGroup
	err = retry(ctx, "update security group", func() (err error) {
		securityGroup, err = c.client.Compute.SecurityGroup.Update(ctx, update)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update security group: %s", err))
		return
	}

	state.FromEntity(securityGroup)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (c computeSecurityGroupResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state computeSecurityGroupResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	err := retryDelete(ctx, "delete security group", func() error {
		return c.client.Compute.SecurityGroup.Delete(ctx, compute.SecurityGroupDeleteReq{ID: uint(state.ID.ValueInt64())})
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete security group: %s", err))
		return
	}
}

func (c computeSecurityGroupResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importStatePassthroughInt64ID(ctx, path.Root("id"), request, response)
}
