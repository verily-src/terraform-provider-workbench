package models

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/sam"
)

type PerimeterModel struct {
	Id               types.String `tfsdk:"id"`
	ResourceId       types.String `tfsdk:"resource_id"`
	Owners           types.Set    `tfsdk:"owners"`
	Users            types.Set    `tfsdk:"users"`
	SyncGoogleGroup  types.Bool   `tfsdk:"sync_google_group"`
	GoogleGroupEmail types.String `tfsdk:"google_group_email"`
}

func NewPerimeterModel(resourceId string, policies []sam.AccessPolicyResponseEntryV2, syncStatus *sam.SyncStatus) *PerimeterModel {
	var ownerEmails, userEmails []string
	for _, p := range policies {
		switch p.PolicyName {
		case "owner":
			ownerEmails = p.Policy.MemberEmails
		case "user":
			userEmails = p.Policy.MemberEmails
		}
	}

	owners := stringSliceToSet(ownerEmails)
	users := stringSliceToSet(userEmails)

	synced := syncStatus != nil
	var groupEmail string
	if synced {
		groupEmail = syncStatus.Email
	}

	return &PerimeterModel{
		Id:               types.StringValue(resourceId),
		ResourceId:       types.StringValue(resourceId),
		Owners:           owners,
		Users:            users,
		SyncGoogleGroup:  types.BoolValue(synced),
		GoogleGroupEmail: types.StringValue(groupEmail),
	}
}

func (m *PerimeterModel) OwnersAsStringSlice(ctx context.Context) ([]string, diag.Diagnostics) {
	return setToStringSlice(ctx, m.Owners)
}

func (m *PerimeterModel) UsersAsStringSlice(ctx context.Context) ([]string, diag.Diagnostics) {
	return setToStringSlice(ctx, m.Users)
}

func stringSliceToSet(s []string) types.Set {
	if len(s) == 0 {
		return types.SetValueMust(types.StringType, []attr.Value{})
	}
	elems := make([]attr.Value, len(s))
	for i, v := range s {
		elems[i] = types.StringValue(v)
	}
	return types.SetValueMust(types.StringType, elems)
}

func setToStringSlice(ctx context.Context, s types.Set) ([]string, diag.Diagnostics) {
	var result []string
	diags := s.ElementsAs(ctx, &result, false)
	return result, diags
}
