package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

type equivalentString struct {
	normalize func(string) string
}

func (m equivalentString) Description(context.Context) string {
	return "Keeps the prior value when the hub treats both forms as equivalent."
}

func (m equivalentString) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m equivalentString) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if known(req.StateValue) && known(req.PlanValue) &&
		m.normalize(req.StateValue.ValueString()) == m.normalize(req.PlanValue.ValueString()) {
		resp.PlanValue = req.StateValue
	}
}

type equivalentSet struct {
	normalize func(string) string
}

func (m equivalentSet) Description(context.Context) string {
	return "Keeps the prior set when the hub treats both forms as equivalent."
}

func (m equivalentSet) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m equivalentSet) PlanModifySet(ctx context.Context, req planmodifier.SetRequest, resp *planmodifier.SetResponse) {
	if equivalentStrings(ctx, req.StateValue, req.PlanValue, m.normalize) {
		resp.PlanValue = req.StateValue
	}
}

// Attribute modifiers can suppress a formatting-only change after the framework
// has already marked computed attributes unknown. Keep the refreshed state if
// no configured attribute needs an update.
func (r *managedResource[T]) ModifyPlan(_ context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || resp.Plan.Raw.IsNull() || len(resp.RequiresReplace) != 0 {
		return
	}

	var planned, prior, config map[string]tftypes.Value
	if err := resp.Plan.Raw.As(&planned); err != nil {
		return
	}
	if err := req.State.Raw.As(&prior); err != nil {
		return
	}
	if err := req.Config.Raw.As(&config); err != nil {
		return
	}
	for name, value := range planned {
		if value.Equal(prior[name]) || (!value.IsKnown() && config[name].IsNull()) {
			continue
		}
		return
	}
	resp.Plan.Raw = req.State.Raw
}
