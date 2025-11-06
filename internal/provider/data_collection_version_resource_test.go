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
)

func TestAccVersionResource(t *testing.T) {
	host := setupFakes(t)

	orgID := uuid.New()
	podID := uuid.New()

	workspaceResourceName := "workbench_data_collection.test"
	workspaceResourceConfig := fmt.Sprintf(`
provider "workbench" {
  host = "%s"
}
resource "workbench_data_collection" "test" {
  display_name = "Test Data Collection"
  user_facing_id = "test-dc"
  description = "terraform-managed-dc"
  organization_id = "%s"
  pod_id = "%s"
  support_email = "testing@example.com"
  organization_name = "verily-test-org"
  therapeutic_tags = ["cardiology", "dermatology"]
  update_frequency = "weekly"
}
`, host, orgID.String(), podID.String())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create data collection first
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
			// Create and Read version resource
			{
				Config: workspaceResourceConfig + testAccVersionResourceConfig(
					"Test Version",
					"terraform-managed-version",
					propertiesString(map[string]string{"key1": "value1", "key3": "value3"}),
					"https://example.com/release-notes",
					true,
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_data_collection_version.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection_version.test",
						tfjsonpath.New("workspace_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection_version.test",
						tfjsonpath.New("display_name"),
						knownvalue.StringExact("Test Version"),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection_version.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("terraform-managed-version"),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection_version.test",
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
						"workbench_data_collection_version.test",
						tfjsonpath.New("release_notes_url"),
						knownvalue.StringExact("https://example.com/release-notes"),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection_version.test",
						tfjsonpath.New("published"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection_version.test",
						tfjsonpath.New("created_by"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection_version.test",
						tfjsonpath.New("created_date"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection_version.test",
						tfjsonpath.New("last_updated_by"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection_version.test",
						tfjsonpath.New("last_updated_date"),
						knownvalue.NotNull(),
					),
				},
			},
			// Update and Read version resource
			{
				Config: workspaceResourceConfig + testAccVersionResourceConfig(
					"Updated Version",
					"updated-description",
					propertiesString(map[string]string{"key1": "value2", "key2": "value2"}),
					"https://example.com/updated-release-notes",
					true,
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_data_collection_version.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection_version.test",
						tfjsonpath.New("display_name"),
						knownvalue.StringExact("Updated Version"),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection_version.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("updated-description"),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection_version.test",
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
					statecheck.ExpectKnownValue(
						"workbench_data_collection_version.test",
						tfjsonpath.New("release_notes_url"),
						knownvalue.StringExact("https://example.com/updated-release-notes"),
					),
				},
			},
			// Import Testing
			{
				ImportState: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					workspaceID, err := getVersionWorkspaceID(s, "test")
					if err != nil {
						return "", err
					}
					versionID, err := getVersionID(s, "test")
					if err != nil {
						return "", err
					}
					return fmt.Sprintf("workspaces/%s/resources/%s", workspaceID, versionID), nil
				},
				ResourceName:      "workbench_data_collection_version.test",
				ImportStateVerify: true,
			},
			// Deletion is handled automatically by the testing framework.
		},
	})
}

func getVersionID(s *terraform.State, versionTFID string) (string, error) {
	rs, ok := s.RootModule().Resources[fmt.Sprintf("workbench_data_collection_version.%s", versionTFID)]
	if !ok {
		return "", fmt.Errorf("data collection version %s not found in state", versionTFID)
	}
	versionID, ok := rs.Primary.Attributes["id"]
	if !ok {
		return "", fmt.Errorf("ID for data collection version %s not found in state", versionTFID)
	}
	return versionID, nil
}

func getVersionWorkspaceID(s *terraform.State, versionTFID string) (string, error) {
	rs, ok := s.RootModule().Resources[fmt.Sprintf("workbench_data_collection_version.%s", versionTFID)]
	if !ok {
		return "", fmt.Errorf("data collection version %s not found in state", versionTFID)
	}
	workspaceID, ok := rs.Primary.Attributes["workspace_id"]
	if !ok {
		return "", fmt.Errorf("workspace ID for data collection version %s not found in state", versionTFID)
	}
	return workspaceID, nil
}

func testAccVersionResourceConfig(displayName string, description string, properties string, releaseNotesURL string, published bool) string {
	return fmt.Sprintf(`
resource "workbench_data_collection_version" "test" {
  workspace_id = workbench_data_collection.test.id
  display_name = "%s"
  description = "%s"
  properties = [
    %s
  ]
  release_notes_url = "%s"
  published = %t
}
`, displayName, description, properties, releaseNotesURL, published)
}
