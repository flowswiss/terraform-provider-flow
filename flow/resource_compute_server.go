package flow

import (
	"context"
	"fmt"

	"github.com/flowswiss/goclient"
	"github.com/flowswiss/goclient/common"
	"github.com/flowswiss/goclient/compute"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ tfsdk.ResourceType            = (*computeServerResourceType)(nil)
	_ tfsdk.Resource                = (*computeServerResource)(nil)
	_ tfsdk.ResourceWithImportState = (*computeServerResource)(nil)
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
}

func (c *computeServerResourceData) FromEntity(server compute.Server) {
	c.ID = types.Int64{Value: int64(server.ID)}
	c.Name = types.String{Value: server.Name}
	c.LocationID = types.Int64{Value: int64(server.Location.ID)}
	c.ImageID = types.Int64{Value: int64(server.Image.ID)}
	c.ProductID = types.Int64{Value: int64(server.Product.ID)}
	c.KeyPairID = types.Int64{Null: server.KeyPair.ID == 0, Value: int64(server.KeyPair.ID)}

	c.NetworkInterfaceID = types.Int64{Null: true}
	if len(server.Networks) != 0 {
		network := server.Networks[0]
		c.NetworkID = types.Int64{Value: int64(network.ID)}
		if len(network.Interfaces) != 0 {
			c.PrivateIP = types.String{Value: network.Interfaces[0].PrivateIP}
			c.NetworkInterfaceID = types.Int64{Value: int64(network.Interfaces[0].ID)}
		}
	}
}

func primaryInterfaceID(server compute.Server) (int, bool) {
	if len(server.Networks) == 0 || len(server.Networks[0].Interfaces) == 0 {
		return 0, false
	}
	return server.Networks[0].Interfaces[0].ID, true
}

type computeServerResourceType struct{}

func (c computeServerResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the server",
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},
			"name": {
				Type:                types.StringType,
				MarkdownDescription: "name of the server",
				Required:            true,
			},
			"location_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the location",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"image_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the image",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
			},
			"product_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the product — changing it resizes the server in place: it is stopped, resized and started again (about a minute of downtime), disks and addresses are kept",
				Required:            true,
			},
			"network_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the initial network (the organisation's default network when omitted)",
				Optional:            true,
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
					tfsdk.UseStateForUnknown(),
				},
			},
			"private_ip": {
				Type:                types.StringType,
				MarkdownDescription: "initial private ip of the server",
				Optional:            true,
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
					tfsdk.UseStateForUnknown(),
				},
			},
			"network_interface_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the server's primary network interface — reference it from elastic ip attachments",
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},
			"security_group_ids": {
				Type:                types.SetType{ElemType: types.Int64Type},
				MarkdownDescription: "security groups on the primary network interface — the organisation's default group when omitted; at least one is required",
				Optional:            true,
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.UseStateForUnknown(),
				},
			},
			"key_pair_id": {
				Type:                types.Int64Type,
				MarkdownDescription: "unique identifier of the key pair (linux images require one)",
				Optional:            true,
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
					tfsdk.UseStateForUnknown(),
				},
			},
			"password": {
				Type:                types.StringType,
				MarkdownDescription: "initial windows password of the server",
				Optional:            true,
				Sensitive:           true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					// TODO: write-only once the framework is on 1.x (Terraform ≥ 1.11 WriteOnly attributes) — until then an imported resource plans a replace here because the api never returns the value
					tfsdk.RequiresReplace(),
				},
			},
			"cloud_init": {
				Type:                types.StringType,
				MarkdownDescription: "cloud init script",
				Optional:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					// TODO: write-only once the framework is on 1.x (Terraform ≥ 1.11 WriteOnly attributes) — until then an imported resource plans a replace here because the api never returns the value
					tfsdk.RequiresReplace(),
				},
			},
		},
	}, nil
}

func (c computeServerResourceType) NewResource(ctx context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	prov, diagnostics := convertToLocalProviderType(p)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	return computeServerResource{
		serverService: compute.NewServerService(prov.client),
		orderService:  common.NewOrderService(prov.client),
	}, diagnostics
}

type computeServerResource struct {
	serverService compute.ServerService
	orderService  common.OrderService
}

func (c computeServerResource) Create(ctx context.Context, request tfsdk.CreateResourceRequest, response *tfsdk.CreateResourceResponse) {
	var config computeServerResourceData
	response.Diagnostics.Append(request.Config.Get(ctx, &config)...)
	if response.Diagnostics.HasError() {
		return
	}

	create := compute.ServerCreate{
		Name:             config.Name.Value,
		LocationID:       int(config.LocationID.Value),
		ImageID:          int(config.ImageID.Value),
		ProductID:        int(config.ProductID.Value),
		AttachExternalIP: false,
		NetworkID:        int(config.NetworkID.Value),
		PrivateIP:        config.PrivateIP.Value,
		KeyPairID:        int(config.KeyPairID.Value),
		Password:         config.Password.Value,
		CloudInit:        config.CloudInit.Value,
	}

	var ordering common.Ordering
	err := retryCreate(ctx, "create server", func() (err error) {
		ordering, err = c.serverService.Create(ctx, create)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create server: %s", err))
		return
	}

	order, err := waitForOrder(ctx, c.orderService, ordering)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for server creation: %s", err))
		return
	}

	server, err := c.waitForServerStatus(ctx, order.Product.ID, compute.ServerStatusRunning, "running")
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for server to be running: %s", err))
		if server.ID == 0 {
			return
		}
	} else if !config.SecurityGroupIDs.Null && !config.SecurityGroupIDs.Unknown {
		if err := c.updateSecurityGroups(ctx, server, config.SecurityGroupIDs); err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update security groups: %s", err))
		}
	}

	var state computeServerResourceData
	state.FromEntity(server)

	state.Password = config.Password
	state.CloudInit = config.CloudInit

	response.Diagnostics.Append(c.readSecurityGroups(ctx, server, &state)...)
	response.Diagnostics.Append(response.State.Set(ctx, state)...)
}

func (c computeServerResource) Read(ctx context.Context, request tfsdk.ReadResourceRequest, response *tfsdk.ReadResourceResponse) {
	var state computeServerResourceData
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	server, err := c.serverService.Get(ctx, int(state.ID.Value))
	if err != nil {
		if isNotFound(err) {
			removeGone(ctx, response, fmt.Sprintf("server %d", state.ID.Value))
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

func (c computeServerResource) Update(ctx context.Context, request tfsdk.UpdateResourceRequest, response *tfsdk.UpdateResourceResponse) {
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

	update := compute.ServerUpdate{
		Name: plan.Name.Value,
	}

	var server compute.Server
	err := retry(ctx, "update server", func() (err error) {
		server, err = c.serverService.Update(ctx, int(state.ID.Value), update)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update server: %s", err))
		return
	}

	if plan.ProductID.Value != state.ProductID.Value {
		resized, err := c.resize(ctx, server, int(plan.ProductID.Value))
		if err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to resize server: %s", err))
			return
		}
		server = resized
	}

	if !plan.SecurityGroupIDs.Unknown && !plan.SecurityGroupIDs.Null && !plan.SecurityGroupIDs.Equal(state.SecurityGroupIDs) {
		if err := c.updateSecurityGroups(ctx, server, plan.SecurityGroupIDs); err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update security groups: %s", err))
			return
		}
	}

	state.FromEntity(server)

	response.Diagnostics.Append(c.readSecurityGroups(ctx, server, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, state)...)
}

func (c computeServerResource) Delete(ctx context.Context, request tfsdk.DeleteResourceRequest, response *tfsdk.DeleteResourceResponse) {
	var state computeServerResourceData
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	err := retryDelete(ctx, "delete server", func() error {
		return c.serverService.Delete(ctx, int(state.ID.Value), false)
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete server: %s", err))
		return
	}
}

func (c computeServerResource) waitForServerStatus(ctx context.Context, serverID int, want int, name string) (server compute.Server, err error) {
	err = waitFor(ctx, serverBootTimeout, defaultWaitInterval, fmt.Sprintf("server %d to be %s", serverID, name), func(ctx context.Context) (bool, error) {
		got, err := c.serverService.Get(ctx, serverID)
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

func (c computeServerResource) ImportState(ctx context.Context, request tfsdk.ImportResourceStateRequest, response *tfsdk.ImportResourceStateResponse) {
	importStatePassthroughInt64ID(ctx, path.Root("id"), request, response)
}

func (c computeServerResource) updateSecurityGroups(ctx context.Context, server compute.Server, groups types.Set) error {
	ifaceID, ok := primaryInterfaceID(server)
	if !ok {
		return fmt.Errorf("server %d has no network interface", server.ID)
	}

	update := compute.NetworkInterfaceSecurityGroupUpdate{SecurityGroupIDs: securityGroupIDs(groups)}
	return retry(ctx, "update security groups", func() (err error) {
		_, err = c.serverService.NetworkInterfaces(server.ID).UpdateSecurityGroups(ctx, ifaceID, update)
		return err
	})
}

func (c computeServerResource) readSecurityGroups(ctx context.Context, server compute.Server, state *computeServerResourceData) (diagnostics diag.Diagnostics) {
	state.SecurityGroupIDs = types.Set{ElemType: types.Int64Type, Null: true}

	ifaceID, ok := primaryInterfaceID(server)
	if !ok {
		return nil
	}

	list, err := c.serverService.NetworkInterfaces(server.ID).List(ctx, goclient.Cursor{NoFilter: 1})
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("unable to list network interfaces of server %d: %s", server.ID, err))
		return diagnostics
	}

	for _, iface := range list.Items {
		if iface.ID == ifaceID {
			state.SecurityGroupIDs = securityGroupIDSet(iface)
			break
		}
	}
	return nil
}

const (
	serverActionStart = "start"
	serverActionStop  = "stop"
)

// the api resizes only a stopped server: stop, upgrade (an order), start —
// a server the user keeps stopped stays stopped
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
		ordering, err = c.serverService.Upgrade(ctx, server.ID, compute.ServerUpgrade{ProductID: productID})
		return err
	})
	if err == nil {
		_, err = waitForOrder(ctx, c.orderService, ordering)
	}
	if err == nil {
		_, err = c.waitForServerStatus(ctx, server.ID, compute.ServerStatusStopped, "stopped")
	}

	if wasStopped {
		if err != nil {
			return server, err
		}
		return c.serverService.Get(ctx, server.ID)
	}

	// started again even after a failed upgrade — a resize must not leave a
	// stopped server behind
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
		_, err = c.serverService.Perform(ctx, serverID, compute.ServerPerform{Action: action})
		return err
	})
}
