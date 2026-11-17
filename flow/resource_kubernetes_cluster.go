package flow

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/flowswiss/goclient/v2/common"
	"github.com/flowswiss/goclient/v2/compute"
	"github.com/flowswiss/goclient/v2/kubernetes"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*kubernetesClusterResource)(nil)
	_ resource.ResourceWithConfigure   = (*kubernetesClusterResource)(nil)
	_ resource.ResourceWithImportState = (*kubernetesClusterResource)(nil)
)

type kubernetesClusterResourceData struct {
	ID   types.Int64  `tfsdk:"id"`
	Name types.String `tfsdk:"name"`

	LocationID      types.Int64 `tfsdk:"location_id"`
	NetworkID       types.Int64 `tfsdk:"network_id"`
	SecurityGroupID types.Int64 `tfsdk:"security_group_id"`

	Public        types.Bool   `tfsdk:"public"`
	PublicAddress types.String `tfsdk:"public_address"`
	DNSName       types.String `tfsdk:"dns_name"`

	VersionID types.Int64 `tfsdk:"version_id"`

	NodeCount     types.Int64 `tfsdk:"node_count"`
	NodeProductID types.Int64 `tfsdk:"node_product_id"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (k *kubernetesClusterResourceData) FromEntity(cluster kubernetes.Cluster) {
	k.ID = types.Int64Value(int64(cluster.ID))
	k.Name = types.StringValue(cluster.Name)

	k.LocationID = types.Int64Value(int64(cluster.Location.ID))
	k.NetworkID = types.Int64Value(int64(cluster.Network.ID))
	k.SecurityGroupID = types.Int64Value(int64(cluster.SecurityGroup.ID))

	if cluster.PublicAddress == "" {
		k.Public = types.BoolValue(false)
		k.PublicAddress = types.StringNull()
	} else {
		k.Public = types.BoolValue(true)
		k.PublicAddress = types.StringValue(cluster.PublicAddress)
	}

	k.DNSName = types.StringValue(cluster.DNSName)

	k.VersionID = types.Int64Value(int64(cluster.Version.ID))

	k.NodeCount = types.Int64Value(int64(cluster.NodeCount.Expected.Worker))
	k.NodeProductID = types.Int64Value(int64(cluster.ExpectedPreset.Worker.ID))
}

func (k kubernetesClusterResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the cluster",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the cluster",
				Required:            true,
			},
			"location_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the location",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"network_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the network",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"security_group_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the security group",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"public": schema.BoolAttribute{
				MarkdownDescription: "indicates if the cluster is public",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"public_address": schema.StringAttribute{
				MarkdownDescription: "public address of the cluster",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"dns_name": schema.StringAttribute{
				MarkdownDescription: "DNS name of the cluster",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the kubernetes version. the platform assigns it on create and it cannot be chosen; setting it on an existing cluster upgrades along the current version's upgrade paths",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"node_count": schema.Int64Attribute{
				MarkdownDescription: "number of nodes in the cluster",
				Required:            true,
			},
			"node_product_id": schema.Int64Attribute{
				MarkdownDescription: "unique identifier of the node product",
				Required:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create:            true,
				CreateDescription: timeoutDescription("bounds the whole create; unset, the order wait is bounded at 10m and the cluster is given 20m to become ready"),
				Update:            true,
				UpdateDescription: timeoutDescription("bounds the whole update; unset, the cluster is given 20m to unlock after each change"),
				Delete:            true,
				DeleteDescription: timeoutDescription("bounds the whole delete; unset, the cluster is given 20m to disappear"),
			}),
		},
	}
}

func newKubernetesClusterResource() resource.Resource {
	return &kubernetesClusterResource{}
}

func (k *kubernetesClusterResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_kubernetes_cluster"
}

func (k *kubernetesClusterResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	client, ok := clientFromProviderData(request.ProviderData, &response.Diagnostics)
	if !ok {
		return
	}

	k.client = client
}

type kubernetesClusterResource struct {
	client flowClient
}

func (k kubernetesClusterResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var config kubernetesClusterResourceData
	diagnostics := request.Config.Get(ctx, &config)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	if !config.VersionID.IsNull() && !config.VersionID.IsUnknown() {
		response.Diagnostics.AddAttributeError(
			path.Root("version_id"),
			"Version Not Settable On Creation",
			"the platform assigns the kubernetes version when the cluster is created; set version_id in a later apply to upgrade an existing cluster along its upgrade paths",
		)
		return
	}

	ctx, cancel := withTimeout(ctx, config.Timeouts.Create, &response.Diagnostics)
	defer cancel()

	create := kubernetes.ClusterCreateReq{
		Name:       config.Name.ValueString(),
		LocationID: int(config.LocationID.ValueInt64()),
		NetworkID:  int(config.NetworkID.ValueInt64()),
		Worker: kubernetes.ClusterWorkerCreateReq{
			ProductID: int(config.NodeProductID.ValueInt64()),
			Count:     int(config.NodeCount.ValueInt64()),
		},
		AttachExternalIP: true,
	}

	if !config.Public.IsNull() && !config.Public.ValueBool() {
		create.AttachExternalIP = false
	}

	var ordering common.Ordering
	err := retryCreate(ctx, "create cluster", func() (err error) {
		ordering, err = k.client.Kubernetes.Cluster.Create(ctx, create)
		return err
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to create cluster: %s", err))
		return
	}

	order, err := waitForOrder(ctx, k.client.Common.Order, ordering)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for cluster creation: %s", err))
		return
	}

	cluster, err := waitForClusterReady(ctx, k.client.Kubernetes.Cluster, order.Product.ID)
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for cluster to be ready: %s", err))
		if cluster.ID == 0 {
			cluster.ID = order.Product.ID
		}
	}

	var state kubernetesClusterResourceData
	state.FromEntity(cluster)
	state.Timeouts = config.Timeouts

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (k kubernetesClusterResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state kubernetesClusterResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	cluster, err := k.client.Kubernetes.Cluster.Get(ctx, kubernetes.ClusterGetReq{ID: uint(state.ID.ValueInt64())})
	if err != nil {
		if isNotFound(err) {
			removeGone(ctx, response, fmt.Sprintf("cluster %d", state.ID.ValueInt64()))
			return
		}
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to get cluster: %s", err))
		return
	}

	state.FromEntity(cluster)

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (k kubernetesClusterResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var state kubernetesClusterResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	var plan kubernetesClusterResourceData
	diagnostics = request.Plan.Get(ctx, &plan)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	ctx, cancel := withTimeout(ctx, plan.Timeouts.Update, &response.Diagnostics)
	defer cancel()

	if plan.Name.ValueString() != state.Name.ValueString() {
		update := kubernetes.ClusterUpdateReq{
			ID:   uint(state.ID.ValueInt64()),
			Name: nonZero(plan.Name.ValueString()),
		}

		err := retry(ctx, "update cluster", func() (err error) {
			_, err = k.client.Kubernetes.Cluster.Update(ctx, update)
			return err
		})
		if err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to update cluster: %s", err))
			return
		}
	}

	if !plan.VersionID.IsUnknown() && plan.VersionID.ValueInt64() != state.VersionID.ValueInt64() {
		if _, err := k.waitForClusterUnlocked(ctx, int(state.ID.ValueInt64())); err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for cluster to be unlocked: %s", err))
			return
		}

		current, err := k.client.Kubernetes.Cluster.GetConfiguration(ctx, kubernetes.ClusterGetReq{ID: uint(state.ID.ValueInt64())})
		if err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to read cluster configuration: %s", err))
			return
		}
		update := kubernetes.ClusterConfigurationReq{
			ID:        uint(state.ID.ValueInt64()),
			VersionID: int(plan.VersionID.ValueInt64()),
			Variables: current.Variables,
		}

		err = retry(ctx, "update cluster configuration", func() (err error) {
			_, err = k.client.Kubernetes.Cluster.UpdateConfiguration(ctx, update)
			return err
		})
		if err != nil {
			if isVariableSchemaMismatch(err) {
				response.Diagnostics.AddAttributeError(
					path.Root("version_id"),
					"Cluster Variables Rejected By The Target Version",
					fmt.Sprintf("the cluster's configuration variables do not fit the schema of kubernetes version %d: %s. terraform does not manage these variables, so adjust them in the portal or run the upgrade there, then apply again", plan.VersionID.ValueInt64(), err),
				)
				return
			}
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to change cluster configuration: %s", err))
			return
		}
	}

	if plan.NodeCount.ValueInt64() != state.NodeCount.ValueInt64() || plan.NodeProductID.ValueInt64() != state.NodeProductID.ValueInt64() {
		if _, err := k.waitForClusterUnlocked(ctx, int(state.ID.ValueInt64())); err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for cluster to be unlocked: %s", err))
			return
		}

		update := kubernetes.ClusterUpdateFlavorReq{
			ID: uint(state.ID.ValueInt64()),
			Worker: kubernetes.ClusterWorkerUpdateReq{
				ProductID: int(plan.NodeProductID.ValueInt64()),
				Count:     int(plan.NodeCount.ValueInt64()),
			},
		}

		err := retry(ctx, "update cluster flavor", func() (err error) {
			_, err = k.client.Kubernetes.Cluster.UpdateFlavor(ctx, update)
			return err
		})
		if err != nil {
			response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to change cluster flavor: %s", err))
			return
		}
	}

	cluster, err := k.waitForClusterUnlocked(ctx, int(state.ID.ValueInt64()))
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for cluster to be unlocked: %s", err))
		return
	}

	state.FromEntity(cluster)
	state.Timeouts = plan.Timeouts

	diagnostics = response.State.Set(ctx, state)
	response.Diagnostics.Append(diagnostics...)
}

func (k kubernetesClusterResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state kubernetesClusterResourceData
	diagnostics := request.State.Get(ctx, &state)
	response.Diagnostics.Append(diagnostics...)
	if response.Diagnostics.HasError() {
		return
	}

	ctx, cancel := withTimeout(ctx, state.Timeouts.Delete, &response.Diagnostics)
	defer cancel()

	err := retryDelete(ctx, "delete cluster", func() error {
		return k.client.Kubernetes.Cluster.Delete(ctx, kubernetes.ClusterDeleteReq{ID: uint(state.ID.ValueInt64())})
	})
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("unable to delete cluster: %s", err))
		return
	}

	if err := k.waitForClusterGone(ctx, int(state.ID.ValueInt64())); err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("waiting for cluster deletion: %s", err))
		return
	}
}

func isVariableSchemaMismatch(err error) bool {
	return statusCode(err) == http.StatusBadRequest && strings.Contains(err.Error(), "invalid property at")
}

func waitForClusterReady(ctx context.Context, service *kubernetes.ClusterService, clusterID int) (cluster kubernetes.Cluster, err error) {
	err = waitFor(ctx, clusterWaitTimeout, defaultWaitInterval, fmt.Sprintf("cluster %d to be ready", clusterID), func(ctx context.Context) (bool, error) {
		got, err := service.Get(ctx, kubernetes.ClusterGetReq{ID: uint(clusterID)})
		if err != nil {
			return false, err
		}
		cluster = got
		return !cluster.Locked && cluster.Status.ID == compute.ClusterStatusHealthy, nil
	})

	return cluster, err
}

func (k kubernetesClusterResource) waitForClusterUnlocked(ctx context.Context, clusterID int) (cluster kubernetes.Cluster, err error) {
	err = waitFor(ctx, clusterWaitTimeout, defaultWaitInterval, fmt.Sprintf("cluster %d to be unlocked", clusterID), func(ctx context.Context) (bool, error) {
		got, err := k.client.Kubernetes.Cluster.Get(ctx, kubernetes.ClusterGetReq{ID: uint(clusterID)})
		if err != nil {
			return false, err
		}
		cluster = got
		return !cluster.Locked, nil
	})

	return cluster, err
}

func (k kubernetesClusterResource) waitForClusterGone(ctx context.Context, clusterID int) error {
	return waitForGone(ctx, clusterWaitTimeout, fmt.Sprintf("cluster %d", clusterID), func(ctx context.Context) error {
		_, err := k.client.Kubernetes.Cluster.Get(ctx, kubernetes.ClusterGetReq{ID: uint(clusterID)})
		return err
	})
}

func (k kubernetesClusterResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	importStatePassthroughInt64ID(ctx, path.Root("id"), request, response)
}
