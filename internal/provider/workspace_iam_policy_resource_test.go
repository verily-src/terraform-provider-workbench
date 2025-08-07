package provider

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/wsm"
)

func TestAccWorkspaceIamPolicyResource(t *testing.T) {
	host := setupFakes(t)
	orgID := uuid.New()
	podID := uuid.New()

	ws1 := withWorkspace("ws1", "test-workspace-1", orgID.String(), podID.String())
	ws2 := withWorkspace("ws2", "test-workspace-2", orgID.String(), podID.String())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccWorkspaceIamPolicyResourceConfig(host, ws1, ws2, "ws1", `
					{
					  role = "READER"
					  members = ["test-user-1", "test-user-2"]
					},
					{
					  role = "WRITER"
					  members = ["test-user-1", "test-user-3"]
					},
				`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_workspace_iam_policy.test",
						tfjsonpath.New("workspace_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace_iam_policy.test",
						tfjsonpath.New("iams"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.MapPartial(map[string]knownvalue.Check{
								"role": knownvalue.StringExact("READER"),
								"members": knownvalue.SetExact([]knownvalue.Check{
									knownvalue.StringExact("test-user-1"),
									knownvalue.StringExact("test-user-2"),
								}),
							}),
							knownvalue.MapPartial(map[string]knownvalue.Check{
								"role": knownvalue.StringExact("WRITER"),
								"members": knownvalue.SetExact([]knownvalue.Check{
									knownvalue.StringExact("test-user-1"),
									knownvalue.StringExact("test-user-3"),
								}),
							}),
						}),
					),
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckWorkspaceRoles(host, "ws1", &wsm.RoleBindingList{
						{
							Role:    wsm.IamRoleREADER,
							Members: &[]string{"test-user-1", "test-user-2"},
						},
						{
							Role:    wsm.IamRoleWRITER,
							Members: &[]string{"test-user-1", "test-user-3"},
						},
					}),
					testAccCheckWorkspaceRoles(host, "ws2", &wsm.RoleBindingList{}),
				),
			},
			// Update and Read testing
			{
				Config: testAccWorkspaceIamPolicyResourceConfig(host, ws1, ws2, "ws1", `
					{
					  role = "READER"
					  members = ["test-user-1", "test-user-3"]
					},
					{
					  role = "OWNER"
					  members = ["test-user-1"]
					},
				`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_workspace_iam_policy.test",
						tfjsonpath.New("workspace_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace_iam_policy.test",
						tfjsonpath.New("iams"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.MapPartial(map[string]knownvalue.Check{
								"role": knownvalue.StringExact("READER"),
								"members": knownvalue.SetExact([]knownvalue.Check{
									knownvalue.StringExact("test-user-1"),
									knownvalue.StringExact("test-user-3"),
								}),
							}),
							knownvalue.MapPartial(map[string]knownvalue.Check{
								"role": knownvalue.StringExact("OWNER"),
								"members": knownvalue.SetExact([]knownvalue.Check{
									knownvalue.StringExact("test-user-1"),
								}),
							}),
						}),
					),
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckWorkspaceRoles(host, "ws1", &wsm.RoleBindingList{
						{
							Role:    wsm.IamRoleREADER,
							Members: &[]string{"test-user-1", "test-user-3"},
						},
						{
							Role:    wsm.IamRoleOWNER,
							Members: &[]string{"test-user-1"},
						},
					}),
					testAccCheckWorkspaceRoles(host, "ws2", &wsm.RoleBindingList{}),
				),
			},
			// Changing workspace ID should cause recreate
			{
				Config: testAccWorkspaceIamPolicyResourceConfig(host, ws1, ws2, "ws2", `
					{
					  role = "READER"
					  members = ["test-user-1", "test-user-3"]
					},
					{
					  role = "OWNER"
					  members = ["test-user-1"]
					},
				`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_workspace_iam_policy.test",
						tfjsonpath.New("workspace_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace_iam_policy.test",
						tfjsonpath.New("iams"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.MapPartial(map[string]knownvalue.Check{
								"role": knownvalue.StringExact("READER"),
								"members": knownvalue.SetExact([]knownvalue.Check{
									knownvalue.StringExact("test-user-1"),
									knownvalue.StringExact("test-user-3"),
								}),
							}),
							knownvalue.MapPartial(map[string]knownvalue.Check{
								"role": knownvalue.StringExact("OWNER"),
								"members": knownvalue.SetExact([]knownvalue.Check{
									knownvalue.StringExact("test-user-1"),
								}),
							}),
						}),
					),
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckWorkspaceRoles(host, "ws1", &wsm.RoleBindingList{}),
					testAccCheckWorkspaceRoles(host, "ws2", &wsm.RoleBindingList{
						{
							Role:    wsm.IamRoleREADER,
							Members: &[]string{"test-user-1", "test-user-3"},
						},
						{
							Role:    wsm.IamRoleOWNER,
							Members: &[]string{"test-user-1"},
						},
					}),
				),
			},
			// Import testing
			{
				ImportState: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					id, err := getWorkspaceID(s, "ws2")
					if err != nil {
						return "", err
					}
					return fmt.Sprintf("workspaces/%s/roles", id), nil
				},
				ResourceName:      "workbench_workspace_iam_policy.test",
				ImportStateVerify: true,
				// This field is used to match the imported resource with the
				// prior state. Since we don't have a unique ID field, just
				// match that the workspace IDs are the same.
				ImportStateVerifyIdentifierAttribute: "workspace_id",
			},
			// Deletion is handled automatically by the testing framework.
		},
	})
}

func testAccWorkspaceIamPolicyResourceConfig(host string, workspace1, workspace2 configModifier, policyWorkspace, iams string) string {
	return generateConfig(
		withProvider(host),
		workspace1,
		workspace2,
		withRaw(fmt.Sprintf(`
resource "workbench_workspace_iam_policy" "test" {
  workspace_id = %s
  iams = [
    %s
  ]
}
`, workspaceByReference(policyWorkspace), iams)))
}
