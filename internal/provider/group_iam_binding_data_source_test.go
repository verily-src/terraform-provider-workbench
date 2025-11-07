package provider

import (
	"testing"

	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/verily-src/terraform-provider-workbench/internal/client"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/user"
)

func TestAccGroupIamBindingDataSource(t *testing.T) {
	orgID := uuid.New()
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == fmt.Sprintf("/api/user/api/groups/v1/%s/members", "test-group") {
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
				Config: testAccGroupIamBindingDataSourceConfig(mockServer.URL, "test-group", orgID.String()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.workbench_group_iam_binding.test-admin", "group", "test-group"),
					resource.TestCheckResourceAttr("data.workbench_group_iam_binding.test-admin", "role", "ADMIN"),
					resource.TestCheckResourceAttr("data.workbench_group_iam_binding.test-admin", "principals.#", "1"),
					resource.TestCheckResourceAttr("data.workbench_group_iam_binding.test-admin", "principals.0.user", "admin@example.com"),

					resource.TestCheckResourceAttr("data.workbench_group_iam_binding.test-member", "group", "test-group"),
					resource.TestCheckResourceAttr("data.workbench_group_iam_binding.test-member", "role", "MEMBER"),
					resource.TestCheckResourceAttr("data.workbench_group_iam_binding.test-member", "principals.#", "2"),
					resource.TestCheckResourceAttr("data.workbench_group_iam_binding.test-member", "principals.0.user", "alice@example.com"),
					resource.TestCheckResourceAttr("data.workbench_group_iam_binding.test-member", "principals.1.group.group", "test-group2"),
					resource.TestCheckResourceAttr("data.workbench_group_iam_binding.test-member", "principals.1.group.organization", orgID.String()),

					resource.TestCheckResourceAttr("data.workbench_group_iam_binding.test-reader", "group", "test-group"),
					resource.TestCheckResourceAttr("data.workbench_group_iam_binding.test-reader", "role", "READER"),
					resource.TestCheckResourceAttr("data.workbench_group_iam_binding.test-reader", "principals.#", "0"),
				),
			},
		},
	})
}

func testAccGroupIamBindingDataSourceConfig(url, groupName, orgID string) string {
	return generateConfig(
		withProvider(url),
		withRaw(fmt.Sprintf(`
data "workbench_group_iam_binding" "test-admin" {
  group        = %q
  organization = %q
  role         = "ADMIN"
}
data "workbench_group_iam_binding" "test-member" {
  group        = %q
  organization = %q
  role         = "MEMBER"
}
data "workbench_group_iam_binding" "test-reader" {
  group        = %q
  organization = %q
  role         = "READER"
}
`, groupName, orgID, groupName, orgID, groupName, orgID)),
	)
}
