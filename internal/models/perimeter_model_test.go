package models

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/sam"
)

func TestNewPerimeterModel(t *testing.T) {
	ownerName := "owner"
	userName := "user"
	syncEmail := "policy-abc@verily-bvdp.com"

	tests := []struct {
		name       string
		resourceId string
		policies   []sam.AccessPolicyResponseEntryV2
		syncStatus *sam.SyncStatus
		wantSync   bool
		wantEmail  string
	}{
		{
			name:       "basic without sync",
			resourceId: "test-perimeter",
			policies: []sam.AccessPolicyResponseEntryV2{
				{
					PolicyName: ownerName,
					Policy: sam.AccessPolicyMembershipV2{
						MemberEmails: []string{"admin@example.com"},
						Roles:        []string{"owner"},
						Actions:      []string{},
					},
				},
				{
					PolicyName: userName,
					Policy: sam.AccessPolicyMembershipV2{
						MemberEmails: []string{"user@example.com"},
						Roles:        []string{"user"},
						Actions:      []string{},
					},
				},
			},
			syncStatus: nil,
			wantSync:   false,
			wantEmail:  "",
		},
		{
			name:       "with sync",
			resourceId: "synced-perimeter",
			policies: []sam.AccessPolicyResponseEntryV2{
				{
					PolicyName: ownerName,
					Policy: sam.AccessPolicyMembershipV2{
						MemberEmails: []string{"admin@example.com"},
						Roles:        []string{"owner"},
						Actions:      []string{},
					},
				},
				{
					PolicyName: userName,
					Policy: sam.AccessPolicyMembershipV2{
						MemberEmails: []string{"user@example.com"},
						Roles:        []string{"user"},
						Actions:      []string{},
					},
				},
			},
			syncStatus: &sam.SyncStatus{
				Email:        syncEmail,
				LastSyncDate: "2024-01-01T00:00:00Z",
			},
			wantSync:  true,
			wantEmail: syncEmail,
		},
		{
			name:       "empty policies",
			resourceId: "empty-perimeter",
			policies:   []sam.AccessPolicyResponseEntryV2{},
			syncStatus: nil,
			wantSync:   false,
			wantEmail:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := NewPerimeterModel(tt.resourceId, tt.policies, tt.syncStatus)

			if got := model.Id.ValueString(); got != tt.resourceId {
				t.Errorf("Id = %q, want %q", got, tt.resourceId)
			}
			if got := model.ResourceId.ValueString(); got != tt.resourceId {
				t.Errorf("ResourceId = %q, want %q", got, tt.resourceId)
			}
			if got := model.SyncGoogleGroup.ValueBool(); got != tt.wantSync {
				t.Errorf("SyncGoogleGroup = %v, want %v", got, tt.wantSync)
			}
			if got := model.GoogleGroupEmail.ValueString(); got != tt.wantEmail {
				t.Errorf("GoogleGroupEmail = %q, want %q", got, tt.wantEmail)
			}
		})
	}
}

func TestPerimeterModel_StringSliceConversion(t *testing.T) {
	ownerName := "owner"
	userName := "user"
	policies := []sam.AccessPolicyResponseEntryV2{
		{
			PolicyName: ownerName,
			Policy: sam.AccessPolicyMembershipV2{
				MemberEmails: []string{"a@example.com", "b@example.com"},
				Roles:        []string{"owner"},
				Actions:      []string{},
			},
		},
		{
			PolicyName: userName,
			Policy: sam.AccessPolicyMembershipV2{
				MemberEmails: []string{"c@example.com"},
				Roles:        []string{"user"},
				Actions:      []string{},
			},
		},
	}

	model := NewPerimeterModel("test", policies, nil)
	ctx := context.Background()

	owners, diags := model.OwnersAsStringSlice(ctx)
	if diags.HasError() {
		t.Fatalf("OwnersAsStringSlice returned errors: %v", diags)
	}
	wantOwners := []string{"a@example.com", "b@example.com"}
	if diff := cmp.Diff(sortStrings(wantOwners), sortStrings(owners)); diff != "" {
		t.Errorf("owners mismatch (-want +got):\n%s", diff)
	}

	users, diags := model.UsersAsStringSlice(ctx)
	if diags.HasError() {
		t.Fatalf("UsersAsStringSlice returned errors: %v", diags)
	}
	wantUsers := []string{"c@example.com"}
	if diff := cmp.Diff(wantUsers, users); diff != "" {
		t.Errorf("users mismatch (-want +got):\n%s", diff)
	}
}

func TestStringSliceToSet(t *testing.T) {
	set := stringSliceToSet([]string{"a", "b"})
	if set.IsNull() {
		t.Error("expected non-null set")
	}
	if !set.ElementType(context.Background()).Equal(types.StringType) {
		t.Error("expected string element type")
	}

	empty := stringSliceToSet(nil)
	if empty.IsNull() {
		t.Error("expected non-null set for nil input")
	}
}

func sortStrings(s []string) []string {
	sorted := make([]string, len(s))
	copy(sorted, s)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	return sorted
}
