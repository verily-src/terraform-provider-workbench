package provider

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/verily-src/terraform-provider-workbench/internal/client"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/wsm"
)

func TestAccWorkspaceResource(t *testing.T) {
	host := setupFakes(t)

	orgID := uuid.New()
	podID := uuid.New()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccWorkspaceResourceConfig(host, orgID.String(), podID.String(), "Test Workspace", "test-workspace", "Managed by terraform", propertiesString(map[string]string{"key1": "value1", "key3": "value3"})),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_workspace.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace.test",
						tfjsonpath.New("user_facing_id"),
						knownvalue.StringExact("test-workspace"),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Managed by terraform"),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace.test",
						tfjsonpath.New("display_name"),
						knownvalue.StringExact("Test Workspace"),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace.test",
						tfjsonpath.New("organization_id"),
						knownvalue.StringExact(orgID.String()),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace.test",
						tfjsonpath.New("pod_id"),
						knownvalue.StringExact(podID.String()),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace.test",
						tfjsonpath.New("created_by"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace.test",
						tfjsonpath.New("created_date"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace.test",
						tfjsonpath.New("last_updated_by"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace.test",
						tfjsonpath.New("last_updated_date"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace.test",
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
						"workbench_workspace.test",
						tfjsonpath.New("location"),
						knownvalue.StringExact("us-central1"),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace.test",
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
				},
			},
			// Update and Read testing
			{
				Config: testAccWorkspaceResourceConfig(host, orgID.String(), podID.String(), "Updated Workspace", "test-workspace", "Updated by terraform", propertiesString(map[string]string{"key1": "value2", "key2": "value2"})),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_workspace.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace.test",
						tfjsonpath.New("user_facing_id"),
						knownvalue.StringExact("test-workspace"),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Updated by terraform"),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace.test",
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
						"workbench_workspace.test",
						tfjsonpath.New("display_name"),
						knownvalue.StringExact("Updated Workspace"),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace.test",
						tfjsonpath.New("organization_id"),
						knownvalue.StringExact(orgID.String()),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace.test",
						tfjsonpath.New("pod_id"),
						knownvalue.StringExact(podID.String()),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace.test",
						tfjsonpath.New("created_by"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace.test",
						tfjsonpath.New("created_date"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace.test",
						tfjsonpath.New("last_updated_by"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_workspace.test",
						tfjsonpath.New("last_updated_date"),
						knownvalue.NotNull(),
					),
				},
			},
			// Import testing
			{
				ImportState:       true,
				ResourceName:      "workbench_workspace.test",
				ImportStateVerify: true,
			},
			// Deletion is handled automatically by the testing framework.
		},
	})
}

func testAccWorkspaceResourceConfig(host string, orgID string, podID string, displayName string, userFacingID string, description string, properties string) string {
	return generateConfig(
		withProvider(host),
		withRaw(fmt.Sprintf(`
resource "workbench_workspace" "test" {
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
}
`, displayName, userFacingID, description, orgID, podID, properties)))
}

func propertiesString(m map[string]string) string {
	var builder strings.Builder

	for k, v := range m {
		builder.WriteString("{\n")
		builder.WriteString(fmt.Sprintf("key = \"%s\"\n", k))
		builder.WriteString(fmt.Sprintf("value = \"%s\"\n", v))
		builder.WriteString("},\n")
	}

	return builder.String()
}

type testPolicy struct {
	Namespace      string
	Name           string
	AdditionalData map[string]string
}

func buildPolicies(policies []testPolicy) *[]wsm.WsmPolicyInput {
	var inputs []wsm.WsmPolicyInput
	for _, p := range policies {
		var additionalData []wsm.WsmPolicyPair
		for k, v := range p.AdditionalData {
			additionalData = append(additionalData, wsm.WsmPolicyPair{Key: client.Ptr(k), Value: client.Ptr(v)})
		}
		inputs = append(inputs, wsm.WsmPolicyInput{
			Namespace:      p.Namespace,
			Name:           p.Name,
			AdditionalData: &additionalData,
		})
	}
	return &inputs
}

func buildProperties(properties map[string]string) *[]wsm.Property {
	var inputs []wsm.Property
	for k, v := range properties {
		inputs = append(inputs, wsm.Property{
			Key:   k,
			Value: v,
		})
	}
	return &inputs
}
