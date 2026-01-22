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

func TestAccDataCollectionResource(t *testing.T) {
	host := setupFakes(t)

	orgID := uuid.New()
	podID := uuid.New()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccDataCollectionResourceConfig(host, orgID.String(), podID.String(), "Test Workspace", "test-workspace", "Managed by terraform", propertiesString(map[string]string{"key1": "value1", "key3": "value3"}), "testing@example.com", "verily-test-org", "[\"cardiology\", \"dermatology\"]", "weekly"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_data_collection.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection.test",
						tfjsonpath.New("user_facing_id"),
						knownvalue.StringExact("test-workspace"),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Managed by terraform"),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection.test",
						tfjsonpath.New("display_name"),
						knownvalue.StringExact("Test Workspace"),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection.test",
						tfjsonpath.New("organization_id"),
						knownvalue.StringExact(orgID.String()),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection.test",
						tfjsonpath.New("pod_id"),
						knownvalue.StringExact(podID.String()),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection.test",
						tfjsonpath.New("created_by"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection.test",
						tfjsonpath.New("created_date"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection.test",
						tfjsonpath.New("last_updated_by"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection.test",
						tfjsonpath.New("last_updated_date"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection.test",
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
						"workbench_data_collection.test",
						tfjsonpath.New("location"),
						knownvalue.StringExact("us-central1"),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection.test",
						tfjsonpath.New("policies"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.MapExact(map[string]knownvalue.Check{
								"namespace": knownvalue.StringExact("terra"),
								"name":      knownvalue.StringExact("exfil-perimeter-constraint"),
								"additional_data": knownvalue.SetExact([]knownvalue.Check{
									knownvalue.MapExact(map[string]knownvalue.Check{
										"key":   knownvalue.StringExact("perimeter-id"),
										"value": knownvalue.StringExact("fake-perimeter-id"),
									}),
								}),
							}),
						}),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection.test",
						tfjsonpath.New("support_email"),
						knownvalue.StringExact("testing@example.com"),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection.test",
						tfjsonpath.New("organization_name"),
						knownvalue.StringExact("verily-test-org"),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection.test",
						tfjsonpath.New("therapeutic_tags"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.StringExact("cardiology"),
							knownvalue.StringExact("dermatology"),
						}),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection.test",
						tfjsonpath.New("update_frequency"),
						knownvalue.StringExact("weekly"),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection.test",
						tfjsonpath.New("gcp_project_id"),
						knownvalue.NotNull(),
					),
				},
			},
			// Update and Read testing
			{
				Config: testAccDataCollectionResourceConfig(host, orgID.String(), podID.String(), "Updated Workspace", "test-workspace", "Updated by terraform", propertiesString(map[string]string{"key1": "value2", "key2": "value2"}), "testing@example.com", "verily-test-org", "[\"cardiology\", \"oncology\"]", "weekly"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_data_collection.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection.test",
						tfjsonpath.New("user_facing_id"),
						knownvalue.StringExact("test-workspace"),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Updated by terraform"),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection.test",
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
						"workbench_data_collection.test",
						tfjsonpath.New("display_name"),
						knownvalue.StringExact("Updated Workspace"),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection.test",
						tfjsonpath.New("organization_id"),
						knownvalue.StringExact(orgID.String()),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection.test",
						tfjsonpath.New("pod_id"),
						knownvalue.StringExact(podID.String()),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection.test",
						tfjsonpath.New("created_by"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection.test",
						tfjsonpath.New("created_date"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection.test",
						tfjsonpath.New("last_updated_by"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection.test",
						tfjsonpath.New("last_updated_date"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection.test",
						tfjsonpath.New("therapeutic_tags"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.StringExact("cardiology"),
							knownvalue.StringExact("oncology"),
						}),
					),
					statecheck.ExpectKnownValue(
						"workbench_data_collection.test",
						tfjsonpath.New("gcp_project_id"),
						knownvalue.NotNull(),
					),
				},
			},
			// Import Testing
			{
				ImportState:       true,
				ResourceName:      "workbench_data_collection.test",
				ImportStateVerify: true,
			},
			// Deletion is handled automatically by the testing framework.
		},
	})
}

func testAccDataCollectionResourceConfig(host string, orgID string, podID string, displayName string, userFacingID string, description string, properties string, supportEmail string, organizationName string, therapeudicTags string, updateFrequency string) string {
	return fmt.Sprintf(`
provider "workbench" {
  host = "%s"
}
resource "workbench_data_collection" "test" {
  display_name = "%s"
  user_facing_id = "%s"
  description = "%s"
  organization_id = "%s"
  pod_id = "%s"
  policies = [
    {
      namespace = "terra"
      name = "exfil-perimeter-constraint"
      additional_data = [
        {
          key = "perimeter-id"
          value = "fake-perimeter-id"
        }
      ]
    }
  ]
  properties = [
    %s
  ]
  support_email = "%s"
  organization_name = "%s"
  therapeutic_tags = %s
  update_frequency = "%s"
}
`, host, displayName, userFacingID, description, orgID, podID, properties, supportEmail, organizationName, therapeudicTags, updateFrequency)
}
