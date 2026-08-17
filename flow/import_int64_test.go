package flow

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func newImportResponse(t *testing.T) *tfsdk.ImportResourceStateResponse {
	t.Helper()
	schema := tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {Type: types.Int64Type, Computed: true},
		},
	}
	return &tfsdk.ImportResourceStateResponse{
		State: tfsdk.State{
			Schema: schema,
			Raw: tftypes.NewValue(schema.TerraformType(context.Background()), map[string]tftypes.Value{
				"id": tftypes.NewValue(tftypes.Number, nil),
			}),
		},
	}
}

func TestImportStatePassthroughInt64ID_Numeric(t *testing.T) {
	ctx := context.Background()
	resp := newImportResponse(t)

	importStatePassthroughInt64ID(ctx, path.Root("id"), tfsdk.ImportResourceStateRequest{ID: "1686"}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	var got types.Int64
	resp.State.GetAttribute(ctx, path.Root("id"), &got)
	if got.Value != 1686 {
		t.Fatalf("expected id=1686, got %d", got.Value)
	}
}

func TestImportStatePassthroughInt64ID_NonNumeric(t *testing.T) {
	ctx := context.Background()
	resp := newImportResponse(t)

	importStatePassthroughInt64ID(ctx, path.Root("id"), tfsdk.ImportResourceStateRequest{ID: "abc"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected an error diagnostic for non-numeric id")
	}
}

// Documents the pre-existing bug: the upstream helper writes a string into an Int64 attribute.
func TestUpstreamPassthroughFailsOnInt64(t *testing.T) {
	ctx := context.Background()
	resp := newImportResponse(t)

	tfsdk.ResourceImportStatePassthroughID(ctx, path.Root("id"), tfsdk.ImportResourceStateRequest{ID: "1686"}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected upstream helper to fail on Int64 attribute")
	}
}
