package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/verily-src/terraform-provider-workbench/internal/api"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/wsm"
)

func TestAccWorkspaceIamMemberResource(t *testing.T) {
	host := setupFakes(t)
	orgID := uuid.New()
	podID := uuid.New()
	ws := withWorkspace("ws", "test-workspace", orgID.String(), podID.String())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccWorkspaceIamMemberResourceConfig(host, ws, "ws", string(wsm.IamRoleWRITER), `"test-user-1"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_workspace_iam_member.test",
						tfjsonpath.New("workspace_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace_iam_member.test",
						tfjsonpath.New("role"),
						knownvalue.StringExact("WRITER"),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace_iam_member.test",
						tfjsonpath.New("members"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.StringExact("test-user-1"),
						}),
					),
				},
			},
			// Update and Read testing
			{
				Config: testAccWorkspaceIamMemberResourceConfig(host, ws, "ws", string(wsm.IamRoleWRITER), `"test-user-2"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_workspace_iam_member.test",
						tfjsonpath.New("workspace_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace_iam_member.test",
						tfjsonpath.New("role"),
						knownvalue.StringExact("WRITER"),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace_iam_member.test",
						tfjsonpath.New("members"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.StringExact("test-user-2"),
						}),
					),
				},
			},
			// Deletion is handled automatically by the testing framework.
		},
	})
}

func TestAccWorkspaceIamMemberResource_nonAuthoritative(t *testing.T) {
	host := setupFakes(t)
	orgID := uuid.New()
	podID := uuid.New()
	ws := withWorkspace("ws", "test-workspace", orgID.String(), podID.String())

	config1 := testAccWorkspaceIamMemberResourceConfig(host, ws, "ws", string(wsm.IamRoleWRITER), `"test-user-1"`)
	config2 := testAccWorkspaceIamMemberResourceConfig(host, ws, "ws", string(wsm.IamRoleWRITER), `"test-user-1", "test-user-3"`)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create the iam member
			{
				Config: config1,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_workspace_iam_member.test",
						tfjsonpath.New("workspace_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace_iam_member.test",
						tfjsonpath.New("role"),
						knownvalue.StringExact("WRITER"),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace_iam_member.test",
						tfjsonpath.New("members"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.StringExact("test-user-1"),
						}),
					),
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckWorkspaceRoles(host, "ws", &wsm.RoleBindingList{
						{
							Role:    wsm.IamRoleWRITER,
							Members: &[]string{"test-user-1"},
						},
					}),
				),
			},
			// Add a second member to the same role externally
			{
				Config: config1,
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						id, err := getWorkspaceID(s, "ws")
						if err != nil {
							return err
						}

						ctx := context.Background()
						return wsmSetRole(ctx, host, id, wsm.GRANT, wsm.IamRoleWRITER, "test-user-2")
					},
					testAccCheckWorkspaceRoles(host, "ws", &wsm.RoleBindingList{
						{
							Role: wsm.IamRoleWRITER,
							Members: &[]string{
								"test-user-1",
								"test-user-2",
							},
						},
					}),
				),
			},
			// Update the members. External changes should not be overwritten.
			{
				Config: config2,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_workspace_iam_member.test",
						tfjsonpath.New("workspace_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace_iam_member.test",
						tfjsonpath.New("role"),
						knownvalue.StringExact("WRITER"),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace_iam_member.test",
						tfjsonpath.New("members"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.StringExact("test-user-1"),
							knownvalue.StringExact("test-user-3"),
						}),
					),
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckWorkspaceRoles(host, "ws", &wsm.RoleBindingList{
						{
							Role: wsm.IamRoleWRITER,
							Members: &[]string{
								"test-user-1",
								"test-user-2",
								"test-user-3",
							},
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
						id, err := getWorkspaceID(s, "ws")
						if err != nil {
							return err
						}

						ctx := context.Background()
						return wsmSetRole(ctx, host, id, wsm.REVOKE, wsm.IamRoleWRITER, "test-user-1")
					},
					testAccCheckWorkspaceRoles(host, "ws", &wsm.RoleBindingList{
						{
							Role: wsm.IamRoleWRITER,
							Members: &[]string{
								"test-user-2",
								"test-user-3",
							},
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
						"workbench_workspace_iam_member.test",
						tfjsonpath.New("workspace_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace_iam_member.test",
						tfjsonpath.New("role"),
						knownvalue.StringExact("WRITER"),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace_iam_member.test",
						tfjsonpath.New("members"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.StringExact("test-user-1"),
							knownvalue.StringExact("test-user-3"),
						}),
					),
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckWorkspaceRoles(host, "ws", &wsm.RoleBindingList{
						{
							Role: wsm.IamRoleWRITER,
							Members: &[]string{
								"test-user-1",
								"test-user-2",
								"test-user-3",
							},
						},
					}),
				),
			},
			// Deletion is handled automatically by the testing framework.
		},
	})
}

func getWorkspaceID(s *terraform.State, workspaceTFID string) (string, error) {
	rs, ok := s.RootModule().Resources[fmt.Sprintf("workbench_workspace.%s", workspaceTFID)]
	if !ok {
		return "", fmt.Errorf("workspace %s not found in state", workspaceTFID)
	}
	workspaceID, ok := rs.Primary.Attributes["id"]
	if !ok {
		return "", fmt.Errorf("ID for workspace %s not found in state", workspaceTFID)
	}
	return workspaceID, nil
}

func wsmSetRole(ctx context.Context, host, workspaceID string, op wsm.SetAccessOperation, role wsm.IamRole, member string) error {
	c, err := api.NewWSMClient(ctx, host, false, "")
	if err != nil {
		return fmt.Errorf("unable to create Workbench client, unexpected error: %v", err)
	}

	err = api.SetRole(ctx, c, workspaceID, wsm.SetAccessRequest{
		Operation: op,
		Role:      role,
		Principal: wsm.Principal{
			UserPrincipal: &wsm.PrincipalUser{
				Email: member,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to set role in workspace: %v", err)
	}
	return nil
}

func testAccWorkspaceIamMemberResourceConfig(host string, workspace configModifier, memberWorkspace, role, members string) string {
	return generateConfig(
		withProvider(host),
		workspace,
		withRaw(fmt.Sprintf(`
resource "workbench_workspace_iam_member" "test" {
  workspace_id = %s
  role = %q
  members = [%s]
}
`, workspaceByReference(memberWorkspace), role, members)))
}
