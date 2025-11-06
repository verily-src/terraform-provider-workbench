package provider

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/verily-src/terraform-provider-workbench/internal/api"
	"github.com/verily-src/terraform-provider-workbench/internal/client"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/user"
)

func TestAccGroupIamMemberResource(t *testing.T) {
	host := setupFakes(t)
	g1 := withGroup("g1", "test-group-1", "test-org-1")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccGroupIamMemberResourceConfig(host, g1, "g1", string(user.GroupRoleMEMBER), []string{`"test-user-1"`}),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_group_iam_member.test",
						tfjsonpath.New("group"),
						knownvalue.StringExact("test-group-1"),
					),
					statecheck.ExpectKnownValue(
						"workbench_group_iam_member.test",
						tfjsonpath.New("role"),
						knownvalue.StringExact("MEMBER"),
					),
					statecheck.ExpectKnownValue(
						"workbench_group_iam_member.test",
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
			},
			// Update and Read testing
			{
				Config: testAccGroupIamMemberResourceConfig(host, g1, "g1", string(user.GroupRoleMEMBER), []string{`"test-user-2"`}),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_group_iam_member.test",
						tfjsonpath.New("group"),
						knownvalue.StringExact("test-group-1"),
					),
					statecheck.ExpectKnownValue(
						"workbench_group_iam_member.test",
						tfjsonpath.New("role"),
						knownvalue.StringExact("MEMBER"),
					),
					statecheck.ExpectKnownValue(
						"workbench_group_iam_member.test",
						tfjsonpath.New("principals"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.MapExact(map[string]knownvalue.Check{
								"user":   knownvalue.StringExact("test-user-2"),
								"group":  knownvalue.Null(),
								"public": knownvalue.Null(),
							}),
						}),
					),
				},
			},
			// Deletion is handled automatically by the testing framework.
		},
	})
}

func TestAccGroupIamMemberResource_nonAuthoritative(t *testing.T) {
	host := setupFakes(t)
	g1 := withGroup("g1", "test-group-1", "test-org-1")

	config1 := testAccGroupIamMemberResourceConfig(host, g1, "g1", string(user.GroupRoleMEMBER), []string{`"test-user-1"`})
	config2 := testAccGroupIamMemberResourceConfig(host, g1, "g1", string(user.GroupRoleMEMBER), []string{`"test-user-1"`, `"test-user-3"`})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create the iam member
			{
				Config: config1,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_group_iam_member.test",
						tfjsonpath.New("group"),
						knownvalue.StringExact("test-group-1"),
					),
					statecheck.ExpectKnownValue(
						"workbench_group_iam_member.test",
						tfjsonpath.New("role"),
						knownvalue.StringExact("MEMBER"),
					),
					statecheck.ExpectKnownValue(
						"workbench_group_iam_member.test",
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
				),
			},
			// Add a second member to the same role externally
			{
				Config: config1,
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						ctx := context.Background()
						orgID, err := getOrgID(s, "g1")
						if err != nil {
							return err
						}
						return userGrantRole(ctx, host, "test-group-1", orgID, user.GRANT, user.GroupRoleMEMBER, "test-user-2")
					},
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
				),
			},
			// Update the members. External changes should not be overwritten.
			{
				Config: config2,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_group_iam_member.test",
						tfjsonpath.New("group"),
						knownvalue.StringExact("test-group-1"),
					),
					statecheck.ExpectKnownValue(
						"workbench_group_iam_member.test",
						tfjsonpath.New("role"),
						knownvalue.StringExact("MEMBER"),
					),
					statecheck.ExpectKnownValue(
						"workbench_group_iam_member.test",
						tfjsonpath.New("principals"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.MapExact(map[string]knownvalue.Check{
								"user":   knownvalue.StringExact("test-user-1"),
								"group":  knownvalue.Null(),
								"public": knownvalue.Null(),
							}),
							knownvalue.MapExact(map[string]knownvalue.Check{
								"user":   knownvalue.StringExact("test-user-3"),
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
						{
							Principal: user.Principal{
								UserPrincipal: &user.PrincipalUser{
									Email: "test-user-3",
								},
							},
							Roles: []user.GroupRole{user.GroupRoleMEMBER},
						},
					}),
				),
			},
			// Remove a terraform member externally
			{
				Config: config2,
				// Since we modified managed members externally, we expect the
				// refresh plan to be non-empty.
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						ctx := context.Background()
						orgID, err := getOrgID(s, "g1")
						if err != nil {
							return err
						}
						return userGrantRole(ctx, host, "test-group-1", orgID, user.REVOKE, user.GroupRoleMEMBER, "test-user-1")
					},
					testAccCheckGroupRoles(host, "test-group-1", "g1", &user.GroupMemberList{
						{
							Principal: user.Principal{
								UserPrincipal: &user.PrincipalUser{
									Email: "test-user-2",
								},
							},
							Roles: []user.GroupRole{user.GroupRoleMEMBER},
						},
						{
							Principal: user.Principal{
								UserPrincipal: &user.PrincipalUser{
									Email: "test-user-3",
								},
							},
							Roles: []user.GroupRole{user.GroupRoleMEMBER},
						},
					}),
				),
			},
			// Reapply the previous config. The externally removed managed
			// member should be re-added.
			{
				Config: config2,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_group_iam_member.test",
						tfjsonpath.New("group"),
						knownvalue.StringExact("test-group-1"),
					),
					statecheck.ExpectKnownValue(
						"workbench_group_iam_member.test",
						tfjsonpath.New("role"),
						knownvalue.StringExact("MEMBER"),
					),
					statecheck.ExpectKnownValue(
						"workbench_group_iam_member.test",
						tfjsonpath.New("principals"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.MapExact(map[string]knownvalue.Check{
								"user":   knownvalue.StringExact("test-user-1"),
								"group":  knownvalue.Null(),
								"public": knownvalue.Null(),
							}),
							knownvalue.MapExact(map[string]knownvalue.Check{
								"user":   knownvalue.StringExact("test-user-3"),
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
						{
							Principal: user.Principal{
								UserPrincipal: &user.PrincipalUser{
									Email: "test-user-3",
								},
							},
							Roles: []user.GroupRole{user.GroupRoleMEMBER},
						},
					}),
				),
			},
			// Deletion is handled automatically by the testing framework.
		},
	})
}

func testAccGroupIamMemberResourceConfig(host string, group configModifier, bindingGroupId, role string, members []string) string {
	return generateConfig(
		withProvider(host),
		group,
		withRaw(fmt.Sprintf(`
resource "workbench_group_iam_member" "test" {
  group = %s
  organization = %s
  role = %q
  principals = [
        %s
  ]
}
`, groupNameByReference(bindingGroupId), groupOrgByReference(bindingGroupId), role, formatUsers(members))))
}

func testAccCheckGroupRoles(host, groupName, groupTFID string, want *user.GroupMemberList) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		orgID, err := getOrgID(s, groupTFID)
		if err != nil {
			return err
		}

		ctx := context.Background()
		c, err := api.NewUserClient(ctx, host, false, "")
		if err != nil {
			return fmt.Errorf("unable to create Workbench client, unexpected error: %v", err)
		}

		got, err := api.GetGroupRoles(ctx, c, groupName, client.Ptr(orgID))
		if err != nil {
			return fmt.Errorf("failed to get IAM bindings for group %s in org %s: %v", groupName, orgID, err)
		}
		for _, r := range *got {
			roles := r.Roles
			slices.Sort(roles)
			r.Roles = roles
		}
		for _, r := range *want {
			roles := r.Roles
			slices.Sort(roles)
			r.Roles = roles
		}

		if diff := cmp.Diff(want, got, cmpopts.SortSlices(func(a, b user.GroupMember) int {
			return strings.Compare(string(a.Principal.UserPrincipal.Email), string(b.Principal.UserPrincipal.Email))
		})); diff != "" {
			return fmt.Errorf("mismatch in IAM bindings for group %s in org %s: (-want +got):\n%s", groupName, orgID, diff)
		}
		return nil
	}
}

func userGrantRole(ctx context.Context, host, groupName, groupOrg string, op user.SetAccessOperation, role user.GroupRole, member string) error {
	c, err := api.NewUserClient(ctx, host, false, "")
	if err != nil {
		return fmt.Errorf("unable to create Workbench client, unexpected error: %v", err)
	}

	err = api.SetGroupRole(ctx, c, groupName, client.Ptr(groupOrg), user.SetAccessRequest{
		Operation: op,
		Role:      user.GroupRole(role),
		Principal: user.Principal{
			UserPrincipal: &user.PrincipalUser{
				Email: member,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to set group role in workspace: %v", err)
	}
	return nil
}

func getOrgID(s *terraform.State, groupTFID string) (string, error) {
	rs, ok := s.RootModule().Resources[fmt.Sprintf("workbench_group.%s", groupTFID)]
	if !ok {
		return "", fmt.Errorf("group %s not found in state", groupTFID)
	}
	orgID, ok := rs.Primary.Attributes["organization_id"]
	if !ok {
		return "", fmt.Errorf("ID for group %s not found in state", groupTFID)
	}
	return orgID, nil
}
