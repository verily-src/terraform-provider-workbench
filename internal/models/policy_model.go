package models

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verily-src/terraform-provider-workbench/internal/client"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/wsm"
)

// PolicyModel is the description of the policy resource.
type PolicyModel struct {
	// Namespace is the policy namespace.
	Namespace types.String `tfsdk:"namespace"`
	// Name is the policy name.
	Name types.String `tfsdk:"name"`
	// AdditionalData is the additional data for the policy.
	AdditionalData *[]AdditionalDataModel `tfsdk:"additional_data"`
}

// AdditionalDataModel is a key-value pair for a policy.
type AdditionalDataModel struct {
	Key   types.String `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
}

// String returns the string representation of the AdditionalDataModel.
func (d *AdditionalDataModel) String() string {
	return d.Key.ValueString() + "=" + d.Value.ValueString()
}

// String returns the string representation of the PolicyModel.
func (p *PolicyModel) String() string {
	s := p.Namespace.ValueString() + "/" + p.Name.ValueString()
	if p.AdditionalData != nil {
		for _, ad := range *p.AdditionalData {
			s += "," + ad.String()
		}
	}
	return s
}

func (p *PolicyModel) compareNameAndNamespace(other *PolicyModel) bool {
	if p == nil && other == nil {
		return true
	}
	if p == nil || other == nil {
		return false
	}
	if p.Namespace.ValueString() != other.Namespace.ValueString() {
		return false
	}
	if p.Name.ValueString() != other.Name.ValueString() {
		return false
	}
	return true
}

func diffAdditionalData(old, new *[]AdditionalDataModel) (deleted []AdditionalDataModel, added []AdditionalDataModel) {
	return diffArrays(old, new, func(o, n AdditionalDataModel) bool {
		return o.Key.ValueString() == n.Key.ValueString() && o.Value.ValueString() == n.Value.ValueString()
	})
}

// DiffPolicies compares the state and plan PolicyModel and returns the policies that were deleted and added.
func DiffPolicies(oldPolicies, newPolicies *[]PolicyModel) (deleted []PolicyModel, added []PolicyModel) {
	deleted, added = diffArrays(oldPolicies, newPolicies, func(old, new PolicyModel) bool {
		return old.compareNameAndNamespace(&new)
	})

	if oldPolicies == nil || newPolicies == nil {
		return deleted, added
	}

	// Detect changes in additional data
	for _, oldP := range *oldPolicies {
		for _, newP := range *newPolicies {
			if oldP.compareNameAndNamespace(&newP) {
				deletedData, addedData := diffAdditionalData(oldP.AdditionalData, newP.AdditionalData)
				if len(deletedData) > 0 {
					deleted = append(deleted, PolicyModel{
						Namespace:      oldP.Namespace,
						Name:           oldP.Name,
						AdditionalData: &deletedData,
					})
				}
				if len(addedData) > 0 {
					added = append(added, PolicyModel{
						Namespace:      newP.Namespace,
						Name:           newP.Name,
						AdditionalData: &addedData,
					})
				}
			}
		}
	}

	return deleted, added
}

// GetPoliciesInput converts the policies input to wsm.WsmPolicyInput.
func GetPoliciesInput(policies *[]PolicyModel) *wsm.WsmPolicyInputs {
	if policies == nil {
		return nil
	}
	var policyModels []wsm.WsmPolicyInput
	for _, p := range *policies {
		var additionalData []wsm.WsmPolicyPair
		for _, ad := range *p.AdditionalData {
			additionalData = append(additionalData, wsm.WsmPolicyPair{
				Key:   client.Ptr(ad.Key.ValueString()),
				Value: client.Ptr(ad.Value.ValueString()),
			})
		}
		policyModels = append(policyModels, wsm.WsmPolicyInput{
			Namespace:      p.Namespace.ValueString(),
			Name:           p.Name.ValueString(),
			AdditionalData: &additionalData,
		})
	}
	return &wsm.WsmPolicyInputs{
		Inputs: policyModels,
	}
}
