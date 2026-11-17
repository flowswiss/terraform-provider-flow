package flow

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func newImportResponse(t *testing.T) *resource.ImportStateResponse {
	t.Helper()
	ctx := context.Background()
	s := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{Computed: true},
		},
	}
	return &resource.ImportStateResponse{
		State: tfsdk.State{
			Schema: s,
			Raw: tftypes.NewValue(s.Type().TerraformType(ctx), map[string]tftypes.Value{
				"id": tftypes.NewValue(tftypes.Number, nil),
			}),
		},
	}
}

func TestImportStatePassthroughInt64ID_Numeric(t *testing.T) {
	ctx := context.Background()
	resp := newImportResponse(t)

	importStatePassthroughInt64ID(ctx, path.Root("id"), resource.ImportStateRequest{ID: "1686"}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	var got types.Int64
	resp.State.GetAttribute(ctx, path.Root("id"), &got)
	if got.ValueInt64() != 1686 {
		t.Fatalf("expected id=1686, got %d", got.ValueInt64())
	}
}

func TestImportStatePassthroughInt64ID_NonNumeric(t *testing.T) {
	ctx := context.Background()
	resp := newImportResponse(t)

	importStatePassthroughInt64ID(ctx, path.Root("id"), resource.ImportStateRequest{ID: "abc"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected an error diagnostic for non-numeric id")
	}
}

func TestUpstreamPassthroughFailsOnInt64(t *testing.T) {
	ctx := context.Background()
	resp := newImportResponse(t)

	resource.ImportStatePassthroughID(ctx, path.Root("id"), resource.ImportStateRequest{ID: "1686"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected upstream helper to fail on Int64 attribute")
	}
}
