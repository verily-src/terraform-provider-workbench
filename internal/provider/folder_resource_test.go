package provider

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccFolderResource(t *testing.T) {
	host := setupFakes(t)

	orgID := uuid.New()
	podID := uuid.New()

	workspaceResourceName := "workbench_workspace.test"
	workspaceResourceConfig := fmt.Sprintf(`
provider "workbench" {
  host = "%s"
}
resource "workbench_workspace" "test" {
  display_name = "Test Workspace"
  description  = "terraform-managed-workspace"
  user_facing_id = "test-workspace"
  organization_id = "%s"
  pod_id = "%s"
}
`, host, orgID.String(), podID.String())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create workspace first
			{
				Config: workspaceResourceConfig,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						workspaceResourceName,
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
				},
			},
			// Create and Read folder resource
			{
				Config: workspaceResourceConfig + testAccFolderResourceConfig(
					"Test Folder",
					"terraform-managed-folder",
					propertiesString(map[string]string{"key1": "value1", "key3": "value3"}),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_folder.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_folder.test",
						tfjsonpath.New("workspace_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_folder.test",
						tfjsonpath.New("display_name"),
						knownvalue.StringExact("Test Folder"),
					),
					statecheck.ExpectKnownValue(
						"workbench_folder.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("terraform-managed-folder"),
					),
					statecheck.ExpectKnownValue(
						"workbench_folder.test",
						tfjsonpath.New("properties"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.MapExact(map[string]knownvalue.Check{
								"key":   knownvalue.StringExact("key1"),
								"value": knownvalue.StringExact("value1"),
							}),
							knownvalue.MapExact(map[string]knownvalue.Check{
								"key":   knownvalue.StringExact("key3"),
								"value": knownvalue.StringExact("value3"),
							}),
						}),
					),
					statecheck.ExpectKnownValue(
						"workbench_folder.test",
						tfjsonpath.New("created_by"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_folder.test",
						tfjsonpath.New("created_date"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_folder.test",
						tfjsonpath.New("last_updated_by"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_folder.test",
						tfjsonpath.New("last_updated_date"),
						knownvalue.NotNull(),
					),
				},
			},
			// Update and Read folder resource
			{
				Config: workspaceResourceConfig + testAccFolderResourceConfig(
					"Updated Folder",
					"updated-description",
					propertiesString(map[string]string{"key1": "value2", "key2": "value2"}),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_folder.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_folder.test",
						tfjsonpath.New("display_name"),
						knownvalue.StringExact("Updated Folder"),
					),
					statecheck.ExpectKnownValue(
						"workbench_folder.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("updated-description"),
					),
					statecheck.ExpectKnownValue(
						"workbench_folder.test",
						tfjsonpath.New("properties"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.MapExact(map[string]knownvalue.Check{
								"key":   knownvalue.StringExact("key1"),
								"value": knownvalue.StringExact("value2"),
							}),
							knownvalue.MapExact(map[string]knownvalue.Check{
								"key":   knownvalue.StringExact("key2"),
								"value": knownvalue.StringExact("value2"),
							}),
						}),
					),
				},
			},
			// Deletion is handled automatically by the testing framework.
		},
	})
}

func testAccFolderResourceConfig(displayName string, description string, properties string) string {
	return fmt.Sprintf(`
resource "workbench_folder" "test" {
  workspace_id = workbench_workspace.test.id
  display_name = "%s"
  description = "%s"
  properties = [
    %s
  ]
}
`, displayName, description, properties)
}
