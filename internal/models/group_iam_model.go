// package models defines the models used in the provider.
package models

import (
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verily-src/terraform-provider-workbench/internal/client"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/user"
)

type GroupIdentifier struct {
	state attr.ValueState
	// The name of the group
	GroupName types.String `tfsdk:"group"`
	// The UxID of the organization. Blank for global groups.
	OrgID types.String `tfsdk:"organization"`
}

func NewGroupIdentifier(groupName string, orgID string) GroupIdentifier {
	orgIDValue := types.StringNull()
	if orgID != "" {
		orgIDValue = types.StringValue(orgID)
	}
	return GroupIdentifier{
		state:     attr.ValueStateKnown,
		GroupName: types.StringValue(groupName),
		OrgID:     orgIDValue,
	}
}

func (g GroupIdentifier) Params() (user.GroupNameParam, *user.OrgIdQueryParam) {
	var orgID *user.OrgIdQueryParam
	if !g.OrgID.IsNull() {
		orgID = client.Ptr(user.OrgId(g.OrgID.ValueString()))
	}
	return user.GroupNameParam(g.GroupName.ValueString()), orgID
}

func (g GroupIdentifier) IsNull() bool {
	return g.state == attr.ValueStateNull
}

func (g GroupIdentifier) Equal(other GroupIdentifier) bool {
	if g.state != other.state {
		return false
	}

	if g.state != attr.ValueStateKnown {
		return true
	}

	return g.GroupName.Equal(other.GroupName) && g.OrgID.Equal(other.OrgID)
}

type GroupPrincipal struct {
	// User is the email of a user.
	User types.String `tfsdk:"user"`
	// Group is the identifier of a group.
	Group *GroupIdentifier `tfsdk:"group"`
	// Public is true if the group is public.
	Public types.Bool `tfsdk:"public"`
}

func convertGroupPrincipalToUm(p GroupPrincipal) user.Principal {
	up := user.Principal{}
	switch {
	case !p.User.IsNull():
		up.UserPrincipal = &user.PrincipalUser{
			Email: p.User.ValueString(),
		}
	case p.Group != nil:
		up.GroupPrincipal = &user.PrincipalWorkbenchGroup{
			GroupName:      p.Group.GroupName.ValueString(),
			OrganizationId: p.Group.OrgID.ValueString(),
		}
	case p.Public.ValueBool():
		up.PublicPrincipal = client.Ptr(true)
	}
	return up
}

// ConvertGroupPrincipalToTf converts a user.Principal to a GroupPrincipal.
func ConvertGroupPrincipalToTf(p user.Principal) GroupPrincipal {
	gp := GroupPrincipal{}
	switch {
	case p.UserPrincipal != nil:
		gp.User = types.StringValue(p.UserPrincipal.Email)
	case p.GroupPrincipal != nil:
		gp.Group = client.Ptr(NewGroupIdentifier(p.GroupPrincipal.GroupName,
			p.GroupPrincipal.OrganizationId))
	case p.GlobalGroupPrincipal != nil:
		gp.Group = client.Ptr(NewGroupIdentifier(p.GlobalGroupPrincipal.GroupName, ""))
	case p.PublicPrincipal != nil && *p.PublicPrincipal:
		gp.Public = types.BoolValue(true)
	}
	return gp
}

// PrincipalMatchesTf checks if a user.Principal matches a GroupPrincipal.
func PrincipalMatchesTf(principal user.Principal, group GroupPrincipal) bool {
	// Match if public
	if group.Public.ValueBool() {
		return principal.PublicPrincipal != nil && *principal.PublicPrincipal
	}

	// Match if user matches
	if group.User.ValueString() != "" {
		if principal.UserPrincipal != nil && principal.UserPrincipal.Email == group.User.ValueString() {
			return true
		}
	}

	// Match if group matches
	if group.Group != nil {
		if principal.GroupPrincipal != nil {
			groupNameMatches := principal.GroupPrincipal.GroupName == group.Group.GroupName.ValueString()
			orgIDMatches := principal.GroupPrincipal.OrganizationId == group.Group.OrgID.ValueString()
			if groupNameMatches && orgIDMatches {
				return true
			}
		}
		if principal.GlobalGroupPrincipal != nil {
			groupNameMatches := principal.GlobalGroupPrincipal.GroupName == group.Group.GroupName.ValueString()
			if groupNameMatches && group.Group.OrgID.ValueString() == "" {
				return true
			}
		}
	}

	return false
}

func (p GroupPrincipal) Equal(other GroupPrincipal) bool {
	return p.User.Equal(other.User) &&
		((p.Group == nil && other.Group == nil) || (p.Group != nil && other.Group != nil && p.Group.Equal(*other.Group))) &&
		p.Public.Equal(other.Public)
}

// GroupRoleBinding is the description of a group role binding.
type GroupRoleBinding struct {
	// Role is the IAM role of the binding.
	Role types.String `tfsdk:"role"`
	// Principals is the list of members in the IAM binding.
	Principals []GroupPrincipal `tfsdk:"principals"`
}

func (b GroupRoleBinding) isMutationOf(other GroupRoleBinding) bool {
	return b.Role.Equal(other.Role)
}

func convertGroupMemberListToBindingMap(members *user.GroupMemberList) map[string]*GroupRoleBinding {
	bindings := make(map[string]*GroupRoleBinding, 0)
	if members == nil {
		return bindings
	}
	for _, member := range *members {
		for _, role := range member.Roles {
			r := string(role)
			b, ok := bindings[r]
			if !ok {
				b = &GroupRoleBinding{
					Role:       types.StringValue(r),
					Principals: make([]GroupPrincipal, 0),
				}
				bindings[r] = b
			}

			b.Principals = append(b.Principals, ConvertGroupPrincipalToTf(member.Principal))
		}
	}
	return bindings
}

func convertGroupMember(members *user.GroupMemberList, role user.GroupRole) *GroupRoleBinding {
	if b := convertGroupMemberListToBindingMap(members)[string(role)]; b != nil {
		return b
	}
	return &GroupRoleBinding{
		Role:       types.StringValue(string(role)),
		Principals: nil,
	}
}

func (b *GroupRoleBinding) toIamBindingModel(group GroupIdentifier) *GroupIamBindingModel {
	return &GroupIamBindingModel{
		GroupIdentifier:  group,
		GroupRoleBinding: *b,
	}
}

// BuildGrantRequests builds a list of SetAccessRequest for granting access to
// members of a role binding.
func (b *GroupRoleBinding) BuildGrantRequests() []user.SetAccessRequest {
	return b.buildSetAccessRequests(user.GRANT)
}

// BuildRevokeRequests builds a list of SetAccessRequest for revoking access
// from members of a role binding.
func (b *GroupRoleBinding) BuildRevokeRequests() []user.SetAccessRequest {
	return b.buildSetAccessRequests(user.REVOKE)
}

func (b *GroupRoleBinding) buildSetAccessRequests(op user.SetAccessOperation) []user.SetAccessRequest {
	if b == nil {
		return nil
	}

	reqs := make([]user.SetAccessRequest, 0)

	for _, p := range b.Principals {
		reqs = append(reqs, user.SetAccessRequest{
			Operation: op,
			Role:      user.GroupRole(b.Role.ValueString()),
			Principal: convertGroupPrincipalToUm(p),

			// We only support access requests without expiration or reason. We
			// may add support for a reason in the future, but expiration is not
			// expected to be supported since this is intended to set permanent
			// IAM bindings.
			Expiration: nil,
			Reason:     nil,
		})
	}

	return reqs
}

// GroupIamBindingModel is the description of a group IAM binding.
type GroupIamBindingModel struct {
	GroupIdentifier
	GroupRoleBinding
}

// NewGroupIamBindingModel creates a new GroupIamBindingModel with the given
// group name, org ID, and members.
func NewGroupIamBindingModel(groupName, orgID string, members *user.GroupMemberList, role user.GroupRole) *GroupIamBindingModel {
	return convertGroupMember(members, role).toIamBindingModel(NewGroupIdentifier(groupName, orgID))
}

func (b GroupIamBindingModel) isMutationOf(other GroupIamBindingModel) bool {
	return b.GroupIdentifier.Equal(other.GroupIdentifier) &&
		b.GroupRoleBinding.isMutationOf(other.GroupRoleBinding)
}

// DiffGroupIamBindings compares the state and plan GroupIamBindingModel and returns
// the members that were deleted and added.
func DiffGroupIamBindings(oldBinding, newBinding *GroupIamBindingModel) (deleted, added *GroupIamBindingModel) {
	if oldBinding == nil && newBinding == nil {
		return nil, nil
	}
	if oldBinding == nil {
		added = newBinding
		return deleted, added
	}
	if newBinding == nil {
		deleted = oldBinding
		return deleted, added
	}
	if !oldBinding.isMutationOf(*newBinding) {
		deleted = oldBinding
		added = newBinding
		return deleted, added
	}

	deletedPrincipals, addedPrincipals := diffArrays(&oldBinding.Principals, &newBinding.Principals, func(oldPrincipal, newPrincipal GroupPrincipal) bool {
		return oldPrincipal.Equal(newPrincipal)
	})

	if len(deletedPrincipals) > 0 {
		deleted = &GroupIamBindingModel{
			GroupIdentifier: oldBinding.GroupIdentifier,
			GroupRoleBinding: GroupRoleBinding{
				Role:       oldBinding.Role,
				Principals: deletedPrincipals,
			},
		}
	}
	if len(addedPrincipals) > 0 {
		added = &GroupIamBindingModel{
			GroupIdentifier: oldBinding.GroupIdentifier,
			GroupRoleBinding: GroupRoleBinding{
				Role:       oldBinding.Role,
				Principals: addedPrincipals,
			},
		}
	}

	return deleted, added
}

// GroupIamPolicyModel is the description of a group iam policy resource.
type GroupIamPolicyModel struct {
	GroupIdentifier
	// Iams is the list of IAM policies attached to the group.
	Iams *[]GroupRoleBinding `tfsdk:"iams"`
}

// NewGroupIamPolicyModel creates a new GroupIamPolicyModel with the given group
// name, org ID, and members.
func NewGroupIamPolicyModel(groupName, groupID string, members *user.GroupMemberList) *GroupIamPolicyModel {
	iamMap := convertGroupMemberListToBindingMap(members)
	iams := make([]GroupRoleBinding, 0, len(iamMap))
	for _, binding := range iamMap {
		iams = append(iams, *binding)
	}

	return (&GroupIamPolicyModel{
		GroupIdentifier: NewGroupIdentifier(groupName, groupID),
		Iams:            &iams,
	}).Normalized()
}

// Normalized removes empty bindings.
func (p *GroupIamPolicyModel) Normalized() *GroupIamPolicyModel {
	if p == nil || p.Iams == nil {
		return p
	}
	newPolicy := *p
	newIams := make([]GroupRoleBinding, 0, len(*p.Iams))
	for _, iam := range *p.Iams {
		if len(iam.Principals) == 0 {
			continue // Skip empty bindings
		}
		newIams = append(newIams, iam)
	}

	slices.SortFunc(newIams, func(a, b GroupRoleBinding) int {
		return strings.Compare(a.Role.ValueString(), b.Role.ValueString())
	})

	newPolicy.Iams = &newIams
	return &newPolicy
}

// BuildGrantRequests builds a list of SetAccessRequest for granting access to
// members of an IAM policy.
func (m *GroupIamPolicyModel) BuildGrantRequests() []user.SetAccessRequest {
	return m.buildSetAccessRequests(user.GRANT)
}

// BuildRevokeRequests builds a list of SetAccessRequest for revoking access
// from members of an IAM policy.
func (m *GroupIamPolicyModel) BuildRevokeRequests() []user.SetAccessRequest {
	return m.buildSetAccessRequests(user.REVOKE)
}

func (m *GroupIamPolicyModel) buildSetAccessRequests(op user.SetAccessOperation) []user.SetAccessRequest {
	if m == nil || m.Iams == nil {
		return nil
	}

	requests := make([]user.SetAccessRequest, 0)
	for _, iam := range *m.Iams {
		reqs := iam.buildSetAccessRequests(op)
		requests = append(requests, reqs...)
	}
	return requests
}

// DiffGroupIamPolicies compares the state and plan GroupIamPolicyModel and returns
// the bindings that were deleted and added.
func DiffGroupIamPolicies(oldPolicy, newPolicy *GroupIamPolicyModel) (deleted, added []GroupRoleBinding) {
	oldPolicy = oldPolicy.Normalized()
	newPolicy = newPolicy.Normalized()

	oldIams := make([]GroupRoleBinding, 0)
	newIams := make([]GroupRoleBinding, 0)
	if oldPolicy != nil && oldPolicy.Iams != nil {
		oldIams = *oldPolicy.Iams
	}
	if newPolicy != nil && newPolicy.Iams != nil {
		newIams = *newPolicy.Iams
	}

	// If either policy has no IAMs, or if the group identifiers are different,
	// then we consider all old IAMs as deleted and all new IAMs as added.
	if len(oldIams) == 0 || len(newIams) == 0 || !oldPolicy.GroupIdentifier.Equal(newPolicy.GroupIdentifier) {
		return oldIams, newIams
	}

	// Any added or missing roles are considered as deleted or added.
	deleted, added = diffArrays(&oldIams, &newIams, func(old, new GroupRoleBinding) bool {
		return old.isMutationOf(new)
	})

	// Detect changes in members
	for _, o := range oldIams {
		for _, n := range newIams {
			if o.isMutationOf(n) {
				deletedBindings, addedBindings := DiffGroupIamBindings(o.toIamBindingModel(oldPolicy.GroupIdentifier), n.toIamBindingModel(newPolicy.GroupIdentifier))
				if deletedBindings != nil {
					deleted = append(deleted, deletedBindings.GroupRoleBinding)
				}
				if addedBindings != nil {
					added = append(added, addedBindings.GroupRoleBinding)
				}
			}
		}
	}

	return deleted, added
}
