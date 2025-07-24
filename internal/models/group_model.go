// package models defines the models used in the provider.
package models

import (
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verily-src/terraform-provider-workbench/internal/client"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/user"
)

// GroupModel is the description of workbench group.
type GroupModel struct {
	// GroupEmail is the email of the group.
	GroupEmail types.String `tfsdk:"group_email"`
	// GroupName is the name of the group.
	GroupName types.String `tfsdk:"group_name"`
	// InternalName is the global name of the group.
	InternalName types.String `tfsdk:"internal_name"`
	// OrgId is the globally unique organization identifier; either UUID or UFID.
	// If it is a UFID, it must be prefixed with a tilde (~).
	OrgId types.String `tfsdk:"organization_id"`
	// OrgUfid is the globally unique user-facing identifier.
	OrgUfid types.String `tfsdk:"organization_user_facing_id"`
	// CreatedBy is the user email of creator.
	CreatedBy types.String `tfsdk:"created_by"`
	// CreatedDate is the timestamp of creation.
	CreatedDate timetypes.RFC3339 `tfsdk:"created_date"`
	// LastUpdatedBy is the user email of last update.
	LastUpdatedBy types.String `tfsdk:"last_updated_by"`
	// LastUpdatedDate is the timestamp of last update.
	LastUpdatedDate timetypes.RFC3339 `tfsdk:"last_updated_date"`
	// ExpirationDays is the number of days until the group expires.
	ExpirationDays types.Int64 `tfsdk:"expiration_days"`
	// ExpirationNotification is whether to notify the user when the group expires.
	ExpirationNotification types.Bool `tfsdk:"expiration_notification"`
	// RequireGrantReason is whether to require a reason for granting access.
	RequireGrantReason types.Bool `tfsdk:"require_grant_reason"`
	// SyncGroup is whether to sync the group with the organization.
	SyncGroup types.Bool `tfsdk:"sync_group"`
	// Description is the description of the group.
	Description types.String `tfsdk:"description"`
}

// NewGroupModel creates a new GroupModel with a given description.
func NewGroupModel(group *user.GroupDescriptionAndRoles) *GroupModel {
	d := group.GroupDescription
	if d == nil {
		groupAndRoles := group.GroupAndRoles
		return &GroupModel{
			GroupEmail:   types.StringValue(groupAndRoles.GroupEmail),
			GroupName:    types.StringValue(groupAndRoles.GroupName),
			InternalName: types.StringPointerValue(groupAndRoles.InternalName),
			OrgId:        types.StringPointerValue(groupAndRoles.OrgId),
			OrgUfid:      types.StringPointerValue(groupAndRoles.OrgUfid),
		}
	}
	return &GroupModel{
		GroupEmail:             types.StringValue(d.GroupEmail),
		GroupName:              types.StringPointerValue(d.GroupName),
		InternalName:           types.StringPointerValue(d.InternalName),
		OrgId:                  types.StringValue(d.OrgId),
		OrgUfid:                types.StringValue(d.OrgUfid),
		CreatedBy:              types.StringValue(d.CreatedBy),
		CreatedDate:            timetypes.NewRFC3339TimeValue(d.CreatedDate),
		LastUpdatedBy:          types.StringValue(d.LastUpdatedBy),
		LastUpdatedDate:        timetypes.NewRFC3339TimeValue(d.LastUpdatedDate),
		ExpirationDays:         types.Int64Value(int64(d.ExpirationDays)),
		ExpirationNotification: types.BoolPointerValue(d.ExpirationNotification),
		RequireGrantReason:     types.BoolPointerValue(d.RequireGrantReason),
		SyncGroup:              types.BoolValue(d.SyncGroup),
		Description:            types.StringPointerValue(d.Description),
	}
}

// ToCreateGroupRequest converts the GroupModel to a CreateGroupRequest.
func (m *GroupModel) ToCreateGroupRequest() user.CreateGroupRequest {
	return user.CreateGroupRequest{
		Description:            m.Description.ValueStringPointer(),
		ExpirationDays:         int64PtrToIntPtr(m.ExpirationDays.ValueInt64Pointer()),
		ExpirationNotification: m.ExpirationNotification.ValueBoolPointer(),
		GroupName:              m.GroupName.ValueString(),
		RequireGrantReason:     m.RequireGrantReason.ValueBoolPointer(),
		SyncGroup:              m.SyncGroup.ValueBoolPointer(),
	}
}

// ToUpdateGroupRequest converts the GroupModel to an UpdateGroupRequest.
func (m *GroupModel) ToUpdateGroupRequest() user.UpdateGroupRequest {
	return user.UpdateGroupRequest{
		Description:            client.Ptr(m.Description.ValueString()),
		Expiration:             int64PtrToIntPtr(m.ExpirationDays.ValueInt64Pointer()),
		ExpirationNotification: m.ExpirationNotification.ValueBoolPointer(),
		RequireGrantReason:     m.RequireGrantReason.ValueBoolPointer(),
	}
}

func int64PtrToIntPtr(p *int64) *int {
	if p == nil {
		return nil
	}
	v := int(*p)
	return &v
}
