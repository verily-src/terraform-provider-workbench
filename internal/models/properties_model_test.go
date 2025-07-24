package models

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"reflect"
	"testing"
)

func TestDiffProperties(t *testing.T) {
	tests := []struct {
		name     string
		old      *[]PropertyModel
		new      *[]PropertyModel
		expected struct {
			deleted []PropertyModel
			added   []PropertyModel
		}
	}{
		{
			name: "no properties",
			old:  nil,
			new:  nil,
			expected: struct {
				deleted []PropertyModel
				added   []PropertyModel
			}{
				deleted: nil,
				added:   nil,
			},
		},
		{
			name: "old properties only",
			old: &[]PropertyModel{
				{
					Key:   types.StringValue("key1"),
					Value: types.StringValue("value1"),
				},
			},
			new: nil,
			expected: struct {
				deleted []PropertyModel
				added   []PropertyModel
			}{
				deleted: []PropertyModel{
					{
						Key:   types.StringValue("key1"),
						Value: types.StringValue("value1"),
					},
				},
				added: nil,
			},
		},
		{
			name: "new properties only",
			old:  nil,
			new: &[]PropertyModel{
				{
					Key:   types.StringValue("key1"),
					Value: types.StringValue("value1"),
				},
			},
			expected: struct {
				deleted []PropertyModel
				added   []PropertyModel
			}{
				deleted: nil,
				added: []PropertyModel{
					{
						Key:   types.StringValue("key1"),
						Value: types.StringValue("value1"),
					},
				},
			},
		},
		{
			name: "properties with same keys and values",
			old: &[]PropertyModel{
				{
					Key:   types.StringValue("key1"),
					Value: types.StringValue("value1"),
				},
				{
					Key:   types.StringValue("key2"),
					Value: types.StringValue("value2"),
				},
			},
			new: &[]PropertyModel{
				{
					Key:   types.StringValue("key2"),
					Value: types.StringValue("value2"),
				},
				{
					Key:   types.StringValue("key1"),
					Value: types.StringValue("value1"),
				},
			},
			expected: struct {
				deleted []PropertyModel
				added   []PropertyModel
			}{
				deleted: nil,
				added:   nil,
			},
		},
		{
			name: "properties with deleted keys",
			old: &[]PropertyModel{
				{
					Key:   types.StringValue("key1"),
					Value: types.StringValue("value1"),
				},
				{
					Key:   types.StringValue("key2"),
					Value: types.StringValue("value2"),
				},
			},
			new: &[]PropertyModel{
				{
					Key:   types.StringValue("key1"),
					Value: types.StringValue("value1"),
				},
			},
			expected: struct {
				deleted []PropertyModel
				added   []PropertyModel
			}{
				deleted: []PropertyModel{
					{
						Key:   types.StringValue("key2"),
						Value: types.StringValue("value2"),
					},
				},
				added: nil,
			},
		},
		{
			name: "properties with same keys but different values",
			old: &[]PropertyModel{
				{
					Key:   types.StringValue("key1"),
					Value: types.StringValue("value1"),
				},
				{
					Key:   types.StringValue("key2"),
					Value: types.StringValue("value2"),
				},
			},
			new: &[]PropertyModel{
				{
					Key:   types.StringValue("key1"),
					Value: types.StringValue("value1"),
				},
				{
					Key:   types.StringValue("key2"),
					Value: types.StringValue("value3"),
				},
			},
			expected: struct {
				deleted []PropertyModel
				added   []PropertyModel
			}{
				deleted: nil,
				added: []PropertyModel{
					{
						Key:   types.StringValue("key2"),
						Value: types.StringValue("value3"),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deleted, added := DiffProperties(tt.old, tt.new)
			if len(deleted) != len(tt.expected.deleted) {
				t.Errorf("DiffProperties() deleted = %v, want %v", deleted, tt.expected.deleted)
			}
			if len(added) != len(tt.expected.added) {
				t.Errorf("DiffProperties() added = %v, want %v", added, tt.expected.added)
			}
		})
	}
}

func TestGetKeys(t *testing.T) {
	tests := []struct {
		name       string
		properties []PropertyModel
		want       []string
	}{
		{
			name:       "no properties",
			properties: []PropertyModel{},
			want:       []string{},
		},
		{
			name: "one property",
			properties: []PropertyModel{
				{
					Key:   types.StringValue("key1"),
					Value: types.StringValue("value1"),
				},
			},
			want: []string{"key1"},
		},
		{
			name: "multiple properties",
			properties: []PropertyModel{
				{
					Key:   types.StringValue("key1"),
					Value: types.StringValue("value1"),
				},
				{
					Key:   types.StringValue("key2"),
					Value: types.StringValue("value2"),
				},
			},
			want: []string{"key1", "key2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetKeys(tt.properties)
			if len(got) != len(tt.want) {
				t.Errorf("GetKeys() = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetKeys() = %v, want %v", got, tt.want)
			}
		})
	}
}
