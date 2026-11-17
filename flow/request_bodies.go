package flow

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func boolPointer(value types.Bool) *bool {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	return new(value.ValueBool())
}

func intPointer(value types.Int64) *int {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	return new(int(value.ValueInt64()))
}

func nonZero[T comparable](value T) *T {
	var zero T
	if value == zero {
		return nil
	}

	return &value
}
