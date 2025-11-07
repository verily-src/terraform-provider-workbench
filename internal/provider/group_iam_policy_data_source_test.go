package provider

import (
	"testing"

	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/verily-src/terraform-provider-workbench/internal/client"
	"github.com/verily-src/terraform-provider-workbench/internal/models"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/user"
)

func TestAccGroupIamPolicyDataSource(t *testing.T) {
	groupName := "test-group"
	orgID := uuid.New()
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == fmt.Sprintf("/api/user/api/groups/v1/%s/members", groupName) {
			desc := []user.GroupMember{
				{
					Principal: user.Principal{
						UserPrincipal: &user.PrincipalUser{
							Email: "alice@example.com",
						},
					},
					Roles: []user.GroupRole{
						user.GroupRoleMEMBER,
					},
				},
				{
					Principal: user.Principal{
						UserPrincipal: &user.PrincipalUser{
							Email: "admin@example.com",
						},
					},
					Roles: []user.GroupRole{
						user.GroupRoleADMIN,
					},
				},
				{
					Principal: user.Principal{
						GroupPrincipal: &user.PrincipalWorkbenchGroup{
							GroupName:      "test-group2",
							OrganizationId: orgID.String(),
						},
					},
					Roles: []user.GroupRole{
						user.GroupRoleMEMBER,
					},
				},
				{
					Principal: user.Principal{
						UserPrincipal: &user.PrincipalUser{
							Email: "support@example.com",
						},
					},
					Roles: []user.GroupRole{
						user.GroupRoleSUPPORT,
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(client.Ptr(desc))
		}
	}))
	defer mockServer.Close()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccGroupIamPolicyDataSourceConfig(mockServer.URL, groupName, orgID.String()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.workbench_group_iam_policy.test", "group", groupName),
					resource.TestCheckResourceAttr("data.workbench_group_iam_policy.test", "organization", orgID.String()),
					resource.TestCheckResourceAttr("data.workbench_group_iam_policy.test", "iams.#", "3"),
					checkIAMHasRoleAndUser(string(user.GroupRoleMEMBER), models.GroupPrincipal{User: types.StringValue("alice@example.com")}),
					checkIAMHasRoleAndUser(string(user.GroupRoleADMIN), models.GroupPrincipal{User: types.StringValue("admin@example.com")}),
					checkIAMHasRoleAndUser(string(user.GroupRoleSUPPORT), models.GroupPrincipal{User: types.StringValue("support@example.com")}),
					checkIAMHasRoleAndUser(string(user.GroupRoleMEMBER), models.GroupPrincipal{Group: &models.GroupIdentifier{GroupName: types.StringValue("test-group2"), OrgID: types.StringValue(orgID.String())}}),
				),
			},
		},
	})
}

func checkIAMHasRoleAndUser(expectedRole string, expectedPrincipal models.GroupPrincipal) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["data.workbench_group_iam_policy.test"]
		if !ok {
			return fmt.Errorf("not found: data.workbench_group_iam_policy.test")
		}

		attrs := rs.Primary.Attributes

		count := 0
		for k := range attrs {
			if strings.HasPrefix(k, "iams.") && strings.HasSuffix(k, ".role") {
				count++
			}
		}
		for i := 0; i < count; i++ {
			roleKey := fmt.Sprintf("iams.%d.role", i)
			if attrs[roleKey] != expectedRole {
				continue
			}

			pCount := 0
			for k := range attrs {
				if strings.HasPrefix(k, fmt.Sprintf("iams.%d.principals.", i)) {
					pCount++
				}
			}

			// Check how many principals are present for this iam
			for j := 0; j < pCount; j++ {
				userKey := fmt.Sprintf("iams.%d.principals.%d.user", i, j)
				groupKey := fmt.Sprintf("iams.%d.principals.%d.group.group", i, j)
				if userVal, ok := attrs[userKey]; ok && userVal == expectedPrincipal.User.ValueString() {
					return nil // Found a match!
				} else if groupVal, ok := attrs[groupKey]; ok && expectedPrincipal.Group != nil &&
					groupVal == expectedPrincipal.Group.GroupName.ValueString() &&
					attrs[fmt.Sprintf("iams.%d.principals.%d.group.organization", i, j)] == expectedPrincipal.Group.OrgID.ValueString() {
					return nil // Found a match!
				}
			}
		}
		return fmt.Errorf("did not find iam with role %q and user %v", expectedRole, expectedPrincipal)
	}
}

func testAccGroupIamPolicyDataSourceConfig(url, groupName, orgID string) string {
	return generateConfig(
		withProvider(url),
		withRaw(fmt.Sprintf(`
data "workbench_group_iam_policy" "test" {
  group        = %q
  organization = %q
}
`, groupName, orgID)),
	)
}
