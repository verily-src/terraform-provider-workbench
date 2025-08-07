package models

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"reflect"
	"testing"
)

func TestDiffPolicies(t *testing.T) {
	tests := []struct {
		name     string
		old      *[]PolicyModel
		new      *[]PolicyModel
		expected struct {
			deleted []PolicyModel
			added   []PolicyModel
		}
	}{
		{
			name: "no policies",
			old:  nil,
			new:  nil,
			expected: struct {
				deleted []PolicyModel
				added   []PolicyModel
			}{
				deleted: nil,
				added:   nil,
			},
		},
		{
			name: "old policies only",
			old: &[]PolicyModel{
				{
					Name:      types.StringValue("policy1"),
					Namespace: types.StringValue("namespace1"),
				},
				{
					Name:      types.StringValue("policy2"),
					Namespace: types.StringValue("namespace2"),
				},
			},
			new: nil,
			expected: struct {
				deleted []PolicyModel
				added   []PolicyModel
			}{
				deleted: []PolicyModel{
					{
						Name:      types.StringValue("policy1"),
						Namespace: types.StringValue("namespace1"),
					},
					{
						Name:      types.StringValue("policy2"),
						Namespace: types.StringValue("namespace2"),
					},
				},
				added: nil,
			},
		},
		{
			name: "new policies only",
			old:  nil,
			new: &[]PolicyModel{
				{
					Name:      types.StringValue("policy3"),
					Namespace: types.StringValue("namespace3"),
				},
				{
					Name:      types.StringValue("policy4"),
					Namespace: types.StringValue("namespace4"),
				},
			},
			expected: struct {
				deleted []PolicyModel
				added   []PolicyModel
			}{
				deleted: nil,
				added: []PolicyModel{
					{
						Name:      types.StringValue("policy3"),
						Namespace: types.StringValue("namespace3"),
					},
					{
						Name:      types.StringValue("policy4"),
						Namespace: types.StringValue("namespace4"),
					},
				},
			},
		},
		{
			name: "some policies deleted and added",
			old: &[]PolicyModel{
				{
					Name:      types.StringValue("policy1"),
					Namespace: types.StringValue("namespace1"),
				},
				{
					Name:      types.StringValue("policy2"),
					Namespace: types.StringValue("namespace2"),
				},
			},
			new: &[]PolicyModel{
				{
					Name:      types.StringValue("policy2"),
					Namespace: types.StringValue("namespace2"),
				},
				{
					Name:      types.StringValue("policy3"),
					Namespace: types.StringValue("namespace3"),
				},
			},
			expected: struct {
				deleted []PolicyModel
				added   []PolicyModel
			}{
				deleted: []PolicyModel{
					{
						Name:      types.StringValue("policy1"),
						Namespace: types.StringValue("namespace1"),
					},
				},
				added: []PolicyModel{
					{
						Name:      types.StringValue("policy3"),
						Namespace: types.StringValue("namespace3"),
					},
				},
			},
		},
		{
			name: "identical policies, list reordered",
			old: &[]PolicyModel{
				{
					Name:      types.StringValue("policy1"),
					Namespace: types.StringValue("namespace1"),
					AdditionalData: &[]AdditionalDataModel{
						{
							Key:   types.StringValue("key"),
							Value: types.StringValue("value1"),
						},
						{
							Key:   types.StringValue("key"),
							Value: types.StringValue("value2"),
						},
						{
							Key:   types.StringValue("key"),
							Value: types.StringValue("value3"),
						},
					},
				},
			},
			new: &[]PolicyModel{
				{
					Name:      types.StringValue("policy1"),
					Namespace: types.StringValue("namespace1"),
					AdditionalData: &[]AdditionalDataModel{
						{
							Key:   types.StringValue("key"),
							Value: types.StringValue("value3"),
						},
						{
							Key:   types.StringValue("key"),
							Value: types.StringValue("value2"),
						},
						{
							Key:   types.StringValue("key"),
							Value: types.StringValue("value1"),
						},
					},
				},
			},
			expected: struct {
				deleted []PolicyModel
				added   []PolicyModel
			}{
				deleted: nil,
				added:   nil,
			},
		},
		{
			name: "new additional data are added",
			old: &[]PolicyModel{
				{
					Name:      types.StringValue("policy1"),
					Namespace: types.StringValue("namespace1"),
					AdditionalData: &[]AdditionalDataModel{
						{
							Key:   types.StringValue("key"),
							Value: types.StringValue("value2"),
						},
					},
				},
			},
			new: &[]PolicyModel{
				{
					Name:      types.StringValue("policy1"),
					Namespace: types.StringValue("namespace1"),
					AdditionalData: &[]AdditionalDataModel{
						{
							Key:   types.StringValue("key"),
							Value: types.StringValue("value2"),
						},
						{
							Key:   types.StringValue("key"),
							Value: types.StringValue("value1"),
						},
					},
				},
			},
			expected: struct {
				deleted []PolicyModel
				added   []PolicyModel
			}{
				added: []PolicyModel{
					{
						Name:      types.StringValue("policy1"),
						Namespace: types.StringValue("namespace1"),
						AdditionalData: &[]AdditionalDataModel{
							{
								Key:   types.StringValue("key"),
								Value: types.StringValue("value1"),
							},
						},
					},
				},
				deleted: nil,
			},
		},
		{
			name: "new additional data are removed",
			old: &[]PolicyModel{
				{
					Name:      types.StringValue("policy1"),
					Namespace: types.StringValue("namespace1"),
					AdditionalData: &[]AdditionalDataModel{
						{
							Key:   types.StringValue("key"),
							Value: types.StringValue("value1"),
						},
						{
							Key:   types.StringValue("key"),
							Value: types.StringValue("value2"),
						},
					},
				},
			},
			new: &[]PolicyModel{
				{
					Name:      types.StringValue("policy1"),
					Namespace: types.StringValue("namespace1"),
					AdditionalData: &[]AdditionalDataModel{
						{
							Key:   types.StringValue("key"),
							Value: types.StringValue("value2"),
						},
					},
				},
			},
			expected: struct {
				deleted []PolicyModel
				added   []PolicyModel
			}{
				deleted: []PolicyModel{
					{
						Name:      types.StringValue("policy1"),
						Namespace: types.StringValue("namespace1"),
						AdditionalData: &[]AdditionalDataModel{
							{
								Key:   types.StringValue("key"),
								Value: types.StringValue("value1"),
							},
						},
					},
				},
				added: nil,
			},
		},
		{
			name: "some additional data are added and removed",
			old: &[]PolicyModel{
				{
					Name:      types.StringValue("policy1"),
					Namespace: types.StringValue("namespace1"),
					AdditionalData: &[]AdditionalDataModel{
						{
							Key:   types.StringValue("key"),
							Value: types.StringValue("value1"),
						},
						{
							Key:   types.StringValue("key"),
							Value: types.StringValue("value2"),
						},
					},
				},
			},
			new: &[]PolicyModel{
				{
					Name:      types.StringValue("policy1"),
					Namespace: types.StringValue("namespace1"),
					AdditionalData: &[]AdditionalDataModel{
						{
							Key:   types.StringValue("key"),
							Value: types.StringValue("value2"),
						},
						{
							Key:   types.StringValue("key"),
							Value: types.StringValue("value3"),
						},
					},
				},
			},
			expected: struct {
				deleted []PolicyModel
				added   []PolicyModel
			}{
				deleted: []PolicyModel{
					{
						Name:      types.StringValue("policy1"),
						Namespace: types.StringValue("namespace1"),
						AdditionalData: &[]AdditionalDataModel{
							{
								Key:   types.StringValue("key"),
								Value: types.StringValue("value1"),
							},
						},
					},
				},
				added: []PolicyModel{
					{
						Name:      types.StringValue("policy1"),
						Namespace: types.StringValue("namespace1"),
						AdditionalData: &[]AdditionalDataModel{
							{
								Key:   types.StringValue("key"),
								Value: types.StringValue("value3"),
							},
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deleted, added := DiffPolicies(test.old, test.new)
			if len(deleted) != len(test.expected.deleted) {
				t.Errorf("expected %d deleted policies, got %d", len(test.expected.deleted), len(deleted))
			}
			if len(deleted) > 0 && !reflect.DeepEqual(deleted, test.expected.deleted) {
				t.Errorf("expected deleted policies %v, got %v", test.expected.deleted, deleted)
			}
			if len(added) != len(test.expected.added) {
				t.Errorf("expected %d added policies, got %d", len(test.expected.added), len(added))
			}
			if len(added) > 0 && !reflect.DeepEqual(added, test.expected.added) {
				t.Errorf("expected added policies %v, got %v", test.expected.added, added)
			}
		})
	}
}
