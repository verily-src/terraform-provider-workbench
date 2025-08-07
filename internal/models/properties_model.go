package models

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/wsm"
)

// PropertyModel is a key-value pair for a workspace.
type PropertyModel struct {
	Key   types.String `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
}

// DiffProperties compares old and new Property input and returns the deleted and added properties.
func DiffProperties(old, new *[]PropertyModel) (deleted []PropertyModel, added []PropertyModel) {
	deleted = make([]PropertyModel, 0)
	added = make([]PropertyModel, 0)
	if old == nil && new == nil {
		return deleted, added
	}
	if old == nil {
		return deleted, *new
	}
	if new == nil {
		return *old, added
	}
	oldMap := make(map[string]PropertyModel)
	newMap := make(map[string]PropertyModel)
	for _, p := range *old {
		oldMap[p.Key.ValueString()] = p
	}
	for _, p := range *new {
		newMap[p.Key.ValueString()] = p
	}
	for k, v := range oldMap {
		if _, ok := newMap[k]; !ok {
			deleted = append(deleted, v)
		}
	}
	for k, v := range newMap {
		if _, ok := oldMap[k]; !ok {
			added = append(added, v)
		} else if v.Value.ValueString() != oldMap[k].Value.ValueString() {
			added = append(added, v)
		}
	}
	return deleted, added
}

// BuildWSMProperties converts a slice of PropertyModel to a slice of wsm.Property.
func BuildWSMProperties(properties *[]PropertyModel) *[]wsm.Property {
	if properties == nil {
		return nil
	}
	wsmProperties := make([]wsm.Property, 0)
	for _, p := range *properties {
		wsmProperties = append(wsmProperties, wsm.Property{
			Key:   p.Key.ValueString(),
			Value: p.Value.ValueString(),
		})
	}
	return &wsmProperties
}

// GetKeys extracts the keys from a slice of PropertyModel.
func GetKeys(properties []PropertyModel) []string {
	keys := make([]string, 0)
	for _, p := range properties {
		keys = append(keys, p.Key.ValueString())
	}
	return keys
}

func convertProperties(properties *[]wsm.Property) *[]PropertyModel {
	if properties == nil {
		return nil
	}
	var propertyModels []PropertyModel
	for _, p := range *properties {
		if strings.HasPrefix(p.Key, "terra-") {
			// Skip service backend managed properties
			continue
		}
		propertyModels = append(propertyModels, PropertyModel{
			Key:   types.StringValue(p.Key),
			Value: types.StringValue(p.Value),
		})
	}
	return &propertyModels
}
