package provider

import (
	"context"
	"errors"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	idPath   = path.Root("id")
	namePath = path.Root("name")
)

func (m *objectID) set(id, name string) error {
	if !validID(id) || name == "" {
		return errors.New("the hub response has an invalid resource ID or name")
	}
	m.ID = types.StringValue(id)
	m.Name = types.StringValue(name)
	return nil
}

func known[T attr.Value](value T) bool {
	return !value.IsNull() && !value.IsUnknown()
}

func stringPtr(value types.String) *string {
	if !known(value) {
		return nil
	}
	return value.ValueStringPointer()
}

func boolPtr(value types.Bool) *bool {
	if !known(value) {
		return nil
	}
	return value.ValueBoolPointer()
}

func intPtr(value types.Int64) *int64 {
	if !known(value) {
		return nil
	}
	return value.ValueInt64Pointer()
}

func optionalStrings(ctx context.Context, value types.Set) (*[]string, error) {
	if !known(value) {
		return nil, nil
	}
	elements := []string{}
	if value.ElementsAs(ctx, &elements, false).HasError() {
		return nil, errors.New("expected a set of known strings")
	}
	return &elements, nil
}

func stringsValue(items *[]string) types.Set {
	var values []attr.Value
	if items != nil {
		values = make([]attr.Value, 0, len(*items))
		for _, item := range *items {
			values = append(values, types.StringValue(item))
		}
	}
	return types.SetValueMust(types.StringType, values)
}

func equivalentStrings(ctx context.Context, first, second types.Set, normalize func(string) string) bool {
	if !known(first) || !known(second) {
		return false
	}

	canonical := func(set types.Set) (types.Set, error) {
		values, err := optionalStrings(ctx, set)
		if err != nil {
			return types.SetNull(types.StringType), err
		}
		normalized := []string{}
		seen := map[string]bool{}
		for _, entry := range *values {
			value := normalize(entry)
			if value != "" && !seen[value] {
				seen[value] = true
				normalized = append(normalized, value)
			}
		}
		return stringsValue(&normalized), nil
	}
	left, leftErr := canonical(first)
	right, rightErr := canonical(second)
	return leftErr == nil && rightErr == nil && left.Equal(right)
}

func validID(id string) bool {
	return id != "" && id != "." && id != ".." && url.PathEscape(id) == id
}
