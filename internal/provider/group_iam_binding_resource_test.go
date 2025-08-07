package provider

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/user"
)

func TestAccGroupIamBindingResource(t *testing.T) {
	host := setupFakes(t)

	g1 := withGroup("g1", "test-group-1", "test-org-1")
	g2 := withGroup("g2", "test-group-2", "test-org-2")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccGroupIamBindingResourceConfig(host, g1, g2, "g1", string(user.GroupRoleMEMBER), []string{`"test-user-1"`}),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_group_iam_binding.test",
						tfjsonpath.New("group"),
						knownvalue.StringExact("test-group-1"),
					),
					statecheck.ExpectKnownValue(
						"workbench_group_iam_binding.test",
						tfjsonpath.New("role"),
						knownvalue.StringExact("MEMBER"),
					),
					statecheck.ExpectKnownValue(
						"workbench_group_iam_binding.test",
						tfjsonpath.New("principals"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.MapExact(map[string]knownvalue.Check{
								"user":   knownvalue.StringExact("test-user-1"),
								"group":  knownvalue.Null(),
								"public": knownvalue.Null(),
							}),
						}),
					),
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupRoles(host, "test-group-1", "g1", &user.GroupMemberList{
						{
							Principal: user.Principal{
								UserPrincipal: &user.PrincipalUser{
									Email: "test-user-1",
								},
							},
							Roles: []user.GroupRole{user.GroupRoleMEMBER},
						},
					}),
					testAccCheckGroupRoles(host, "test-group-2", "g2", &user.GroupMemberList{}),
				),
			},
			// Update and Read testing
			{
				Config: testAccGroupIamBindingResourceConfig(host, g1, g2, "g1", string(user.GroupRoleMEMBER), []string{`"test-user-1"`, `"test-user-2"`}),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_group_iam_binding.test",
						tfjsonpath.New("group"),
						knownvalue.StringExact("test-group-1"),
					),
					statecheck.ExpectKnownValue(
						"workbench_group_iam_binding.test",
						tfjsonpath.New("role"),
						knownvalue.StringExact("MEMBER"),
					),
					statecheck.ExpectKnownValue(
						"workbench_group_iam_binding.test",
						tfjsonpath.New("principals"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.MapExact(map[string]knownvalue.Check{
								"user":   knownvalue.StringExact("test-user-1"),
								"group":  knownvalue.Null(),
								"public": knownvalue.Null(),
							}),
							knownvalue.MapExact(map[string]knownvalue.Check{
								"user":   knownvalue.StringExact("test-user-2"),
								"group":  knownvalue.Null(),
								"public": knownvalue.Null(),
							}),
						}),
					),
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupRoles(host, "test-group-1", "g1", &user.GroupMemberList{
						{
							Principal: user.Principal{
								UserPrincipal: &user.PrincipalUser{
									Email: "test-user-1",
								},
							},
							Roles: []user.GroupRole{user.GroupRoleMEMBER},
						},
						{
							Principal: user.Principal{
								UserPrincipal: &user.PrincipalUser{
									Email: "test-user-2",
								},
							},
							Roles: []user.GroupRole{user.GroupRoleMEMBER},
						},
					}),
					testAccCheckGroupRoles(host, "test-group-2", "g2", &user.GroupMemberList{}),
				),
			},
			// change group will remove the binding from group 1 and create the binding in group 2.
			{
				Config: testAccGroupIamBindingResourceConfig(host, g1, g2, "g2", string(user.GroupRoleMEMBER), []string{`"test-user-1"`, `"test-user-2"`}),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_group_iam_binding.test",
						tfjsonpath.New("group"),
						knownvalue.StringExact("test-group-2"),
					),
					statecheck.ExpectKnownValue(
						"workbench_group_iam_binding.test",
						tfjsonpath.New("role"),
						knownvalue.StringExact("MEMBER"),
					),
					statecheck.ExpectKnownValue(
						"workbench_group_iam_binding.test",
						tfjsonpath.New("principals"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.MapExact(map[string]knownvalue.Check{
								"user":   knownvalue.StringExact("test-user-1"),
								"group":  knownvalue.Null(),
								"public": knownvalue.Null(),
							}),
							knownvalue.MapExact(map[string]knownvalue.Check{
								"user":   knownvalue.StringExact("test-user-2"),
								"group":  knownvalue.Null(),
								"public": knownvalue.Null(),
							}),
						}),
					),
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupRoles(host, "test-group-2", "g2", &user.GroupMemberList{
						{
							Principal: user.Principal{
								UserPrincipal: &user.PrincipalUser{
									Email: "test-user-1",
								},
							},
							Roles: []user.GroupRole{user.GroupRoleMEMBER},
						},
						{
							Principal: user.Principal{
								UserPrincipal: &user.PrincipalUser{
									Email: "test-user-2",
								},
							},
							Roles: []user.GroupRole{user.GroupRoleMEMBER},
						},
					}),
					testAccCheckGroupRoles(host, "test-group-1", "g1", &user.GroupMemberList{}),
				),
			},
			// Import testing
			{
				ImportState: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					orgId, err := getOrgID(s, "g2")
					if err != nil {
						return "", err
					}
					groupName := "test-group-2"
					return fmt.Sprintf("organizations/%s/groups/%s/roles/MEMBER", orgId, groupName), nil
				},
				ResourceName:      "workbench_group_iam_binding.test",
				ImportStateVerify: true,
				// This field is used to match the imported resource with the
				// prior state. Since we don't have a unique ID field, just
				// match that the group IDs are the same.
				ImportStateVerifyIdentifierAttribute: "group",
			},
			// Deletion is handled automatically by the testing framework.
		},
	})
}

func testAccGroupIamBindingResourceConfig(host string, group1, group2 configModifier, bindingGroupId, role string, members []string) string {
	return generateConfig(
		withProvider(host),
		group1,
		group2,
		withRaw(fmt.Sprintf(`
resource "workbench_group_iam_binding" "test" {
  group = %s
  organization = %s
  role = %q
  principals = [
        %s
  ]
}
`, groupNameByReference(bindingGroupId), groupOrgByReference(bindingGroupId), role, formatUsers(members))))
}

func formatUsers(users []string) string {
	var sb strings.Builder
	for i, user := range users {
		sb.WriteString(fmt.Sprintf(`{ user = %s}`, user))
		if i < len(users)-1 {
			sb.WriteString(",\n")
		} else {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
