package provider

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/verily-src/terraform-provider-workbench/internal/api"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/wsm"
)

func TestAccWorkspaceIamBindingResource(t *testing.T) {
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
				Config: testAccWorkspaceIamBindingResourceConfig(host, ws1, ws2, "ws1", string(wsm.IamRoleWRITER), `"test-user-1"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_workspace_iam_binding.test",
						tfjsonpath.New("workspace_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace_iam_binding.test",
						tfjsonpath.New("role"),
						knownvalue.StringExact("WRITER"),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace_iam_binding.test",
						tfjsonpath.New("members"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.StringExact("test-user-1"),
						}),
					),
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckWorkspaceRoles(host, "ws1", &wsm.RoleBindingList{
						{
							Role:    wsm.IamRoleWRITER,
							Members: &[]string{"test-user-1"},
						},
					}),
					testAccCheckWorkspaceRoles(host, "ws2", &wsm.RoleBindingList{}),
				),
			},
			// Update and Read testing
			{
				Config: testAccWorkspaceIamBindingResourceConfig(host, ws1, ws2, "ws1", string(wsm.IamRoleWRITER), `"test-user-1", "test-user-2"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_workspace_iam_binding.test",
						tfjsonpath.New("workspace_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace_iam_binding.test",
						tfjsonpath.New("role"),
						knownvalue.StringExact("WRITER"),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace_iam_binding.test",
						tfjsonpath.New("members"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.StringExact("test-user-1"),
							knownvalue.StringExact("test-user-2"),
						}),
					),
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckWorkspaceRoles(host, "ws1", &wsm.RoleBindingList{
						{
							Role:    wsm.IamRoleWRITER,
							Members: &[]string{"test-user-1", "test-user-2"},
						},
					}),
					testAccCheckWorkspaceRoles(host, "ws2", &wsm.RoleBindingList{}),
				),
			},
			// Changing workspace ID should cause recreate
			{
				Config: testAccWorkspaceIamBindingResourceConfig(host, ws1, ws2, "ws2", string(wsm.IamRoleWRITER), `"test-user-1", "test-user-2"`),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_workspace_iam_binding.test",
						tfjsonpath.New("workspace_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace_iam_binding.test",
						tfjsonpath.New("role"),
						knownvalue.StringExact("WRITER"),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace_iam_binding.test",
						tfjsonpath.New("members"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.StringExact("test-user-1"),
							knownvalue.StringExact("test-user-2"),
						}),
					),
				},
				Check: resource.ComposeTestCheckFunc(
					testAccCheckWorkspaceRoles(host, "ws1", &wsm.RoleBindingList{}),
					testAccCheckWorkspaceRoles(host, "ws2", &wsm.RoleBindingList{
						{
							Role:    wsm.IamRoleWRITER,
							Members: &[]string{"test-user-1", "test-user-2"},
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
					return fmt.Sprintf("workspaces/%s/roles/WRITER", id), nil
				},
				ResourceName:      "workbench_workspace_iam_binding.test",
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

func testAccCheckWorkspaceRoles(host, workspaceTFID string, want *wsm.RoleBindingList) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		workspaceID, err := getWorkspaceID(s, workspaceTFID)
		if err != nil {
			return err
		}

		ctx := context.Background()
		c, err := api.NewWSMClient(ctx, host, false)
		if err != nil {
			return fmt.Errorf("unable to create Workbench client, unexpected error: %v", err)
		}

		got, err := api.GetRoles(ctx, c, workspaceID)
		if err != nil {
			return fmt.Errorf("failed to get IAM bindings for workspace %s: %v", workspaceTFID, err)
		}

		if diff := cmp.Diff(want, got, cmpopts.SortSlices(strings.Compare), cmpopts.SortSlices(func(a, b wsm.RoleBinding) int {
			return strings.Compare(string(a.Role), string(b.Role))
		})); diff != "" {
			return fmt.Errorf("mismatch in IAM bindings for workspace %s: (-want +got):\n%s", workspaceTFID, diff)
		}
		return nil
	}
}

func testAccWorkspaceIamBindingResourceConfig(host string, workspace1, workspace2 configModifier, bindingWorkspace, role, members string) string {
	return generateConfig(
		withProvider(host),
		workspace1,
		workspace2,
		withRaw(fmt.Sprintf(`
resource "workbench_workspace_iam_binding" "test" {
  workspace_id = %s
  role = %q
  members = [%s]
}
`, workspaceByReference(bindingWorkspace), role, members)))
}
