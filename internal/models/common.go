// package models defines the models used in the provider.
package models

import (
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// convertStringArray converts a slice of strings to a slice of types.String.
func convertStringArray(array *[]string) *[]types.String {
	if array == nil {
		return nil
	}
	values := make([]types.String, 0, len(*array))
	for _, v := range *array {
		values = append(values, types.StringValue(v))
	}
	return &values
}

// diffArrays compares two potentially nil slices and returns deleted and added
// elements based on the provided equality function.
func diffArrays[T any](oldArray, newArray *[]T, equals func(old, new T) bool) (deleted []T, added []T) {
	deleted = make([]T, 0)
	added = make([]T, 0)
	if oldArray == nil && newArray == nil {
		return deleted, added
	}
	if oldArray == nil {
		return deleted, *newArray
	}
	if newArray == nil {
		return *oldArray, added
	}
	// Detect removals
	for _, o := range *oldArray {
		found := false
		for _, n := range *newArray {
			if equals(o, n) {
				found = true
				break
			}
		}
		if !found {
			deleted = append(deleted, o)
		}
	}
	// Detect additions
	for _, n := range *newArray {
		found := false
		for _, o := range *oldArray {
			if equals(n, o) {
				found = true
				break
			}
		}
		if !found {
			added = append(added, n)
		}
	}
	return deleted, added
}

func uuidToStringType(uuid *uuid.UUID) types.String {
	if uuid == nil {
		return types.StringNull()
	}
	return types.StringValue(uuid.String())
}

func parseUuid(uuidStr types.String) *uuid.UUID {
	uuidValue, err := uuid.Parse(uuidStr.ValueString())
	if err != nil {
		return nil
	}
	return &uuidValue
}
