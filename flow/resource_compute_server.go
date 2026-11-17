package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient/v2/common"
	"github.com/flowswiss/goclient/v2/compute"
	"github.com/flowswiss/goclient/v2/core"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*computeServerResource)(nil)
	_ resource.ResourceWithConfigure   = (*computeServerResource)(nil)
	_ resource.ResourceWithImportState = (*computeServerResource)(nil)
)

type computeServerResourceData struct {
	ID         types.Int64  `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	LocationID types.Int64  `tfsdk:"location_id"`
	ImageID    types.Int64  `tfsdk:"image_id"`
	ProductID  types.Int64  `tfsdk:"product_id"`
	NetworkID  types.Int64  `tfsdk:"network_id"`
	PrivateIP  types.String `tfsdk:"private_ip"`
	KeyPairID  types.Int64  `tfsdk:"key_pair_id"`
	Password   types.String `tfsdk:"password"`
	CloudInit  types.String `tfsdk:"cloud_init"`

	NetworkInterfaceID types.Int64 `tfsdk:"network_interface_id"`
	SecurityGroupIDs   types.Set   `tfsdk:"security_group_ids"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (c *computeServerResourceData) FromEntity(server compute.Server) {
	c.ID = types.Int64Value(int64(server.ID))
	c.Name = types.StringValue(server.Name)
	c.LocationID = types.Int64Value(int64(server.Location.ID))
	c.ImageID = types.Int64Value(int64(server.Image.ID))
	c.ProductID = types.Int64Value(int64(server.Product.ID))
	c.KeyPairID = types.Int64Null()
	if server.KeyPair.ID != 0 {
		c.KeyPairID = types.Int64Value(int64(server.KeyPair.ID))
	}

	c.NetworkID = types.Int64Null()
	c.PrivateIP = types.StringNull()
	c.NetworkInterfaceID = types.Int64Null()
	if len(server.Networks) != 0 {
		network := server.Networks[0]
		c.NetworkID = types.Int64Value(int64(network.ID))
		if len(network.Interfaces) != 0 {
			c.PrivateIP = types.StringValue(network.Interfaces[0].PrivateIP)
			c.NetworkInterfaceID = types.Int64Value(int64(network.Interfaces[0].ID))
		}
	}
}

func primaryInterfaceID(server compute.Server) (int, bool) {
	if len(server.Networks) == 0 || len(server.Networks[0].Interfaces) == 0 {
		return 0, false
	}
	return server.Networks[0].Interfaces[0].ID, true
}

func (c computeServerResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the server",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the server",
				Required:            true,
			},
			"location_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the location",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"image_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the image",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"product_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the product — changing it resizes the server in place: it is stopped, resized and started again (about a minute of downtime), disks and addresses are kept",
				Required:            true,
			},
			"network_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the initial network (the organisation's default network when omitted)",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"private_ip": schema.StringAttribute{
				MarkdownDescription: "initial private ip of the server",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"network_interface_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the server's primary network interface — reference it from elastic ip attachments",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"security_group_ids": schema.SetAttribute{
				ElementType:         types.Int64Type,
				MarkdownDescription: "security groups on the primary network interface — the organisation's default group when omitted; at least one is required",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			"key_pair_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the key pair (linux images require one)",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "initial windows password of the server; editing it produces no plan, rotate with `terraform apply -replace=`",
				Optional:            true,
				Sensitive:           true,
				WriteOnly:           true,
			},
			"cloud_init": schema.StringAttribute{
				MarkdownDescription: "cloud init script; editing it produces no plan, rotate with `terraform apply -replace=`",
				Optional:            true,
				WriteOnly:           true,
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create:            true,
				CreateDescription: timeoutDescription("bounds the whole create; unset, the order wait and the wait for the server to boot are bounded at 10m each"),
				Update:            true,
				UpdateDescription: timeoutDescription("bounds the whole update; unset, a resize is bounded at 10m per step — stop, the upgrade order, back to stopped, start — so up to 40m"),
				Delete:            true,
				DeleteDescription: timeoutDescription("bounds the whole delete; unset, the server is given 10m to disappear"),
			}),
		},
	}
}

func newComputeServerResource() resource.Resource {
	return &computeServerResource{}
}

func (c *computeServerResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_compute_server"
}

func (c *computeServerResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	c.client = client
}

type computeServerResource struct {
	client flowClient
}

func (c computeServerResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var config computeServerResourceData
	response.Diagnostics.Append(request.Plan.Get(ctx, &config)...)
	if response.Diagnostics.HasError() {
		return
	}

	var password, cloudInit types.String
	response.Diagnostics.Append(request.Config.GetAttribute(ctx, path.Root("password"), &password)...)
	response.Diagnostics.Append(request.Config.GetAttribute(ctx, path.Root("cloud_init"), &cloudInit)...)
	if response.Diagnostics.HasError() {
		return
	}

	ctx, cancel := withTimeout(ctx, config.Timeouts.Create, &response.Diagnostics)
	defer cancel()

	create := compute.ServerCreateReq{
		Name:             config.Name.ValueString(),
		LocationID:       int(config.LocationID.ValueInt64()),
		ImageID:          int(config.ImageID.ValueInt64()),
		ProductID:        int(config.ProductID.ValueInt64()),
		AttachExternalIP: false,
		NetworkID:        int(config.NetworkID.ValueInt64()),
		PrivateIP:        nonZero(config.PrivateIP.ValueString()),
		KeyPairID:        nonZero(int(config.KeyPairID.ValueInt64())),
		Password:         nonZero(password.ValueString()),
		CloudInit:        nonZero(cloudInit.ValueString()),
	}

	var ordering common.Ordering
	err := retryCreate(ctx, "create server", func() (err error) {
		ordering, err = c.client.Compute.Server.Create(ctx, create)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create server: %s", err))
		return
	}

	order, err := waitForOrder(ctx, c.client.Common.Order, ordering)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for server creation: %s", err))
		return
	}

	server, err := c.waitForServerStatus(ctx, order.Product.ID, compute.ServerStatusRunning, "running")
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for server to be running: %s", err))
		// Return only if the server was never fetched. Otherwise it exists and is
		// saved below with the error, so Terraform replaces it on the next apply
		// instead of forgetting it.
		if server.ID == 0 {
			server.ID = order.Product.ID
		}
	} else if !config.SecurityGroupIDs.IsNull() && !config.SecurityGroupIDs.IsUnknown() {
		if err := c.updateSecurityGroups(ctx, server, config.SecurityGroupIDs); err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update security groups: %s", err))
		}
	}

	var state computeServerResourceData
	state.FromEntity(server)

	state.Timeouts = config.Timeouts

	response.Diagnostics.Append(c.readSecurityGroups(ctx, server, &state)...)
	response.Diagnostics.Append(response.State.Set(ctx, state)...)
}

func (c computeServerResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state computeServerResourceData
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	server, err := c.client.Compute.Server.Get(ctx, compute.ServerGetReq{ID: uint(state.ID.ValueInt64())})
	if err != nil {
		if isNotFound(err) {
			removeGone(ctx, response, fmt.Sprintf("server %d", state.ID.ValueInt64()))
			return
		}
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get server: %s", err))
		return
	}

	state.FromEntity(server)

	response.Diagnostics.Append(c.readSecurityGroups(ctx, server, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, state)...)
}

func (c computeServerResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var state computeServerResourceData
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	var plan computeServerResourceData
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}

	ctx, cancel := withTimeout(ctx, plan.Timeouts.Update, &response.Diagnostics)
	defer cancel()

	update := compute.ServerUpdateReq{
		ID:   uint(state.ID.ValueInt64()),
		Name: plan.Name.ValueString(),
	}

	var server compute.Server
	err := retry(ctx, "update server", func() (err error) {
		server, err = c.client.Compute.Server.Update(ctx, update)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update server: %s", err))
		return
	}

	if plan.ProductID.ValueInt64() != state.ProductID.ValueInt64() {
		resized, err := c.resize(ctx, server, int(plan.ProductID.ValueInt64()))
		if err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to resize server: %s", err))
			return
		}
		server = resized
	}

	if !plan.SecurityGroupIDs.IsUnknown() && !plan.SecurityGroupIDs.IsNull() && !plan.SecurityGroupIDs.Equal(state.SecurityGroupIDs) {
		if err := c.updateSecurityGroups(ctx, server, plan.SecurityGroupIDs); err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update security groups: %s", err))
			return
		}
	}

	if fresh, err := c.client.Compute.Server.Get(ctx, compute.ServerGetReq{ID: uint(state.ID.ValueInt64())}); err != nil {
		response.Diagnostics.AddWarning(
			"Incomplete Read",
			fmt.Sprintf("server %d could not be read back after the update: %s", state.ID.ValueInt64(), err),
		)
	} else {
		server = fresh
	}

	state.FromEntity(server)
	state.Timeouts = plan.Timeouts

	response.Diagnostics.Append(c.readSecurityGroups(ctx, server, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, state)...)
}

func (c computeServerResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state computeServerResourceData
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	ctx, cancel := withTimeout(ctx, state.Timeouts.Delete, &response.Diagnostics)
	defer cancel()

	serverID := int(state.ID.ValueInt64())

	err := retryDelete(ctx, "delete server", func() error {
		return c.client.Compute.Server.Delete(ctx, compute.ServerDeleteReq{ID: uint(serverID)})
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete server: %s", err))
		return
	}

	err = waitForGone(ctx, goneTimeout, fmt.Sprintf("server %d", serverID), func(ctx context.Context) error {
		_, err := c.client.Compute.Server.Get(ctx, compute.ServerGetReq{ID: uint(serverID)})
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for server deletion: %s", err))
		return
	}
}

func (c computeServerResource) waitForServerStatus(ctx context.Context, serverID int, want int, name string) (server compute.Server, err error) {
	err = waitFor(ctx, serverBootTimeout, defaultWaitInterval, fmt.Sprintf("server %d to be %s", serverID, name), func(ctx context.Context) (bool, error) {
		got, err := c.client.Compute.Server.Get(ctx, compute.ServerGetReq{ID: uint(serverID)})
		if err != nil {
			return false, err
		}
		server = got

		switch server.Status.ID {
		case want:
			return true, nil
		case compute.ServerStatusError:
			return false, fmt.Errorf("server %d is in error state", serverID)
		default:
			return false, nil
		}
	})

	return server, err
}

func (c computeServerResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importStatePassthroughInt64ID(ctx, path.Root("id"), request, response)
}

func (c computeServerResource) updateSecurityGroups(ctx context.Context, server compute.Server, groups types.Set) error {
	ifaceID, ok := primaryInterfaceID(server)
	if !ok {
		return fmt.Errorf("server %d has no network interface", server.ID)
	}

	update := compute.NetworkInterfaceSecurityGroupUpdateReq{
		ServerID:           uint(server.ID),
		NetworkInterfaceID: uint(ifaceID),
		SecurityGroupIDs:   securityGroupIDs(groups),
	}
	return retry(ctx, "update security groups", func() (err error) {
		_, err = c.client.Compute.NetworkInterface.UpdateSecurityGroups(ctx, update)
		return err
	})
}

func (c computeServerResource) readSecurityGroups(ctx context.Context, server compute.Server, state *computeServerResourceData) (diagnostics diag.Diagnostics) {
	state.SecurityGroupIDs = types.SetNull(types.Int64Type)

	ifaceID, ok := primaryInterfaceID(server)
	if !ok {
		diagnostics.AddWarning(
			"Security Groups Unknown",
			fmt.Sprintf("server %d reports no network interface, so its security groups could not be read", server.ID),
		)
		return diagnostics
	}

	list, err := c.client.Compute.NetworkInterface.List(ctx, compute.NetworkInterfaceListReq{ServerID: uint(server.ID), Cursor: core.CursorAll})
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("unable to list network interfaces of server %d: %s", server.ID, err))
		return diagnostics
	}

	for _, iface := range list.Items {
		if iface.ID == ifaceID {
			state.SecurityGroupIDs = securityGroupIDSet(iface)
			return diagnostics
		}
	}

	diagnostics.AddWarning(
		"Security Groups Unknown",
		fmt.Sprintf("network interface %d is not in the interface list of server %d, so its security groups could not be read", ifaceID, server.ID),
	)
	return diagnostics
}

const (
	serverActionStart = "start"
	serverActionStop  = "stop"
)

func (c computeServerResource) resize(ctx context.Context, server compute.Server, productID int) (compute.Server, error) {
	wasStopped := server.Status.ID == compute.ServerStatusStopped
	if !wasStopped {
		if err := c.perform(ctx, server.ID, serverActionStop); err != nil {
			return server, err
		}
		if _, err := c.waitForServerStatus(ctx, server.ID, compute.ServerStatusStopped, "stopped"); err != nil {
			return server, err
		}
	}

	var ordering common.Ordering
	err := retry(ctx, "upgrade server", func() (err error) {
		ordering, err = c.client.Compute.Server.Upgrade(ctx, compute.ServerUpgradeReq{ID: uint(server.ID), ProductID: productID})
		return err
	})
	if err == nil {
		_, err = waitForOrder(ctx, c.client.Common.Order, ordering)
	}
	if err == nil {
		_, err = c.waitForServerStatus(ctx, server.ID, compute.ServerStatusStopped, "stopped")
	}

	if wasStopped {
		if err != nil {
			return server, err
		}
		return c.client.Compute.Server.Get(ctx, compute.ServerGetReq{ID: uint(server.ID)})
	}

	if startErr := c.perform(ctx, server.ID, serverActionStart); startErr != nil && err == nil {
		err = startErr
	}
	running, waitErr := c.waitForServerStatus(ctx, server.ID, compute.ServerStatusRunning, "running")
	if err != nil {
		return running, err
	}
	return running, waitErr
}

func (c computeServerResource) perform(ctx context.Context, serverID int, action string) error {
	return retry(ctx, action+" server", func() (err error) {
		_, err = c.client.Compute.Server.Perform(ctx, compute.ServerPerformReq{ID: uint(serverID), Action: action})
		return err
	})
}
