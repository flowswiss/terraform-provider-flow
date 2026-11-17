package flow

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func normalizeSecondsDuration(value string) (string, error) {
	duration, err := time.ParseDuration(value)
	if err != nil {
		return "", fmt.Errorf("unable to parse %q as a duration: %w", value, err)
	}

	if duration < time.Second {
		return "", fmt.Errorf("%q is below the one second the api can store", value)
	}

	return duration.Truncate(time.Second).String(), nil
}

type secondsDurationModifier struct{}

func normalizedSecondsDuration() planmodifier.String {
	return secondsDurationModifier{}
}

func (s secondsDurationModifier) Description(ctx context.Context) string {
	return s.MarkdownDescription(ctx)
}

func (s secondsDurationModifier) MarkdownDescription(context.Context) string {
	return "normalizes the duration to whole seconds, as the api reports it back"
}

func (s secondsDurationModifier) PlanModifyString(ctx context.Context, request planmodifier.StringRequest, response *planmodifier.StringResponse) {
	if request.PlanValue.IsNull() || request.PlanValue.IsUnknown() {
		return
	}

	normalized, err := normalizeSecondsDuration(request.PlanValue.ValueString())
	if err != nil {
		return
	}

	response.PlanValue = types.StringValue(normalized)
}

type secondsDurationValidator struct{}

func secondsDuration() validator.String {
	return secondsDurationValidator{}
}

func (s secondsDurationValidator) Description(ctx context.Context) string {
	return s.MarkdownDescription(ctx)
}

func (s secondsDurationValidator) MarkdownDescription(context.Context) string {
	return "a duration of at least one second, such as `10s` or `2m`"
}

func (s secondsDurationValidator) ValidateString(ctx context.Context, request validator.StringRequest, response *validator.StringResponse) {
	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() {
		return
	}

	if _, err := normalizeSecondsDuration(request.ConfigValue.ValueString()); err != nil {
		response.Diagnostics.AddAttributeError(request.Path, "Invalid Duration", err.Error())
	}
}
