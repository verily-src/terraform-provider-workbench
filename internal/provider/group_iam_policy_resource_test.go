package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/user"
)

func TestAccGroupIamPolicyResource(t *testing.T) {
	host := setupFakes(t)

	g1 := withGroup("g1", "test-group-1", "test-org-1")
	g2 := withGroup("g2", "test-group-2", "test-org-2")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccGroupIamPolicyResourceConfig(host, g1, g2, "g1", `
					{
					  role = "ADMIN"
					  principals = [
					  {
					    user = "test-user-1"
					  },
					  {
					    user = "test-user-2"
					  }
					  ]
					},
					{
					  role = "MEMBER"
					  principals = [
					  {
					    user = "test-user-1"
					  },
					  {
					    user = "test-user-3"
					  }
					  ]
					},
				`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_group_iam_policy.test",
						tfjsonpath.New("group"),
						knownvalue.StringExact("test-group-1"),
					),
					statecheck.ExpectKnownValue(
						"workbench_group_iam_policy.test",
						tfjsonpath.New("iams"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.MapPartial(map[string]knownvalue.Check{
								"role": knownvalue.StringExact("ADMIN"),
								"principals": knownvalue.SetExact([]knownvalue.Check{
									knownvalue.MapPartial(map[string]knownvalue.Check{
										"user": knownvalue.StringExact("test-user-1"),
									}),
									knownvalue.MapPartial(map[string]knownvalue.Check{
										"user": knownvalue.StringExact("test-user-2"),
									}),
								}),
							}),
							knownvalue.MapPartial(map[string]knownvalue.Check{
								"role": knownvalue.StringExact("MEMBER"),
								"principals": knownvalue.SetExact([]knownvalue.Check{
									knownvalue.MapPartial(map[string]knownvalue.Check{
										"user": knownvalue.StringExact("test-user-1"),
									}),
									knownvalue.MapPartial(map[string]knownvalue.Check{
										"user": knownvalue.StringExact("test-user-3"),
									}),
								}),
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
							Roles: []user.GroupRole{user.GroupRoleMEMBER, user.GroupRoleADMIN},
						},
						{
							Principal: user.Principal{
								UserPrincipal: &user.PrincipalUser{
									Email: "test-user-2",
								},
							},
							Roles: []user.GroupRole{user.GroupRoleADMIN},
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
					testAccCheckGroupRoles(host, "test-group-2", "g2", &user.GroupMemberList{}),
				),
			},
			// Update and Read testing
			{
				Config: testAccGroupIamPolicyResourceConfig(host, g1, g2, "g1", `
					{
					  role = "ADMIN"
					  principals = [
						{
						  user = "test-user-1"
						},
						{
						  user = "test-user-3"
						}
					  ]
					},
					{
					  role = "MEMBER"
					  principals = [
						{
						  user = "test-user-1"
						}
					  ]
					},
				`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_group_iam_policy.test",
						tfjsonpath.New("group"),
						knownvalue.StringExact("test-group-1"),
					),
					statecheck.ExpectKnownValue(
						"workbench_group_iam_policy.test",
						tfjsonpath.New("iams"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.MapPartial(map[string]knownvalue.Check{
								"role": knownvalue.StringExact("ADMIN"),
								"principals": knownvalue.SetExact([]knownvalue.Check{
									knownvalue.MapPartial(map[string]knownvalue.Check{
										"user": knownvalue.StringExact("test-user-1"),
									}),
									knownvalue.MapPartial(map[string]knownvalue.Check{
										"user": knownvalue.StringExact("test-user-3"),
									}),
								}),
							}),
							knownvalue.MapPartial(map[string]knownvalue.Check{
								"role": knownvalue.StringExact("MEMBER"),
								"principals": knownvalue.SetExact([]knownvalue.Check{
									knownvalue.MapPartial(map[string]knownvalue.Check{
										"user": knownvalue.StringExact("test-user-1"),
									}),
								}),
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
							Roles: []user.GroupRole{user.GroupRoleMEMBER, user.GroupRoleADMIN},
						},
						{
							Principal: user.Principal{
								UserPrincipal: &user.PrincipalUser{
									Email: "test-user-3",
								},
							},
							Roles: []user.GroupRole{user.GroupRoleADMIN},
						},
					}),
					testAccCheckGroupRoles(host, "test-group-2", "g2", &user.GroupMemberList{}),
				),
			},
			// Changing group ID should cause recreate
			{
				Config: testAccGroupIamPolicyResourceConfig(host, g1, g2, "g2", `
					{
					  role = "ADMIN"
					  principals = [
						{
						  user = "test-user-1"
						},
						{
						  user = "test-user-3"
						}
					  ]
					},
					{
					  role = "MEMBER"
					  principals = [
						{
						  user = "test-user-1"
						}
					  ]
					},
				`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_group_iam_policy.test",
						tfjsonpath.New("group"),
						knownvalue.StringExact("test-group-2"),
					),
					statecheck.ExpectKnownValue(
						"workbench_group_iam_policy.test",
						tfjsonpath.New("iams"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.MapPartial(map[string]knownvalue.Check{
								"role": knownvalue.StringExact("ADMIN"),
								"principals": knownvalue.SetExact([]knownvalue.Check{
									knownvalue.MapPartial(map[string]knownvalue.Check{
										"user": knownvalue.StringExact("test-user-1"),
									}),
									knownvalue.MapPartial(map[string]knownvalue.Check{
										"user": knownvalue.StringExact("test-user-3"),
									}),
								}),
							}),
							knownvalue.MapPartial(map[string]knownvalue.Check{
								"role": knownvalue.StringExact("MEMBER"),
								"principals": knownvalue.SetExact([]knownvalue.Check{
									knownvalue.MapPartial(map[string]knownvalue.Check{
										"user": knownvalue.StringExact("test-user-1"),
									}),
								}),
							}),
						}),
					),
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupRoles(host, "test-group-1", "g1", &user.GroupMemberList{}),
					testAccCheckGroupRoles(host, "test-group-2", "g2", &user.GroupMemberList{
						{
							Principal: user.Principal{
								UserPrincipal: &user.PrincipalUser{
									Email: "test-user-1",
								},
							},
							Roles: []user.GroupRole{user.GroupRoleMEMBER, user.GroupRoleADMIN},
						},
						{
							Principal: user.Principal{
								UserPrincipal: &user.PrincipalUser{
									Email: "test-user-3",
								},
							},
							Roles: []user.GroupRole{user.GroupRoleADMIN},
						},
					}),
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
				ResourceName:      "workbench_group_iam_policy.test",
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

func testAccGroupIamPolicyResourceConfig(host string, group1, group2 configModifier, bindingGroupId, iams string) string {
	return generateConfig(
		withProvider(host),
		group1,
		group2,
		withRaw(fmt.Sprintf(`
resource "workbench_group_iam_policy" "test" {
  group = %s
  organization = %s
  iams = [
    %s
  ]
}
`, groupNameByReference(bindingGroupId), groupOrgByReference(bindingGroupId), iams)))
}
