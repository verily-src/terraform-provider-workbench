package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccPerimeterResource(t *testing.T) {
	host := setupFakes(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccPerimeterResourceConfig(host, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_perimeter.test",
						tfjsonpath.New("resource_id"),
						knownvalue.StringExact("acme-research"),
					),
					statecheck.ExpectKnownValue(
						"workbench_perimeter.test",
						tfjsonpath.New("id"),
						knownvalue.StringExact("acme-research"),
					),
					statecheck.ExpectKnownValue(
						"workbench_perimeter.test",
						tfjsonpath.New("sync_google_group"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"workbench_perimeter.test",
						tfjsonpath.New("google_group_email"),
						knownvalue.StringExact(""),
					),
					statecheck.ExpectKnownValue(
						"workbench_perimeter.test",
						tfjsonpath.New("owners"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.StringExact("admin@example.com"),
						}),
					),
					statecheck.ExpectKnownValue(
						"workbench_perimeter.test",
						tfjsonpath.New("users"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.StringExact("user@example.com"),
						}),
					),
				},
			},
			// Update and Read testing - add members and enable sync
			{
				Config: testAccPerimeterResourceConfigUpdated(host),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_perimeter.test",
						tfjsonpath.New("sync_google_group"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"workbench_perimeter.test",
						tfjsonpath.New("google_group_email"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_perimeter.test",
						tfjsonpath.New("owners"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.StringExact("admin@example.com"),
							knownvalue.StringExact("admin2@example.com"),
						}),
					),
					statecheck.ExpectKnownValue(
						"workbench_perimeter.test",
						tfjsonpath.New("users"),
						knownvalue.SetExact([]knownvalue.Check{
							knownvalue.StringExact("user@example.com"),
							knownvalue.StringExact("user2@example.com"),
						}),
					),
				},
			},
			// Import testing
			{
				ResourceName:                         "workbench_perimeter.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "resource_id",
			},
			// Deletion is handled automatically by the testing framework.
		},
	})
}

func TestAccPerimeterResource_WithSyncFromStart(t *testing.T) {
	host := setupFakes(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPerimeterResourceConfigSynced(host),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_perimeter.test",
						tfjsonpath.New("sync_google_group"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"workbench_perimeter.test",
						tfjsonpath.New("google_group_email"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccPerimeterResource_SyncCannotBeDisabled(t *testing.T) {
	host := setupFakes(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPerimeterResourceConfigSynced(host),
			},
			{
				Config:      testAccPerimeterResourceConfig(host, false),
				ExpectError: regexp.MustCompile(`Sync cannot be disabled`),
			},
		},
	})
}

func testAccPerimeterResourceConfig(host string, syncGoogleGroup bool) string {
	return fmt.Sprintf(`
provider "workbench" {
  host = "%s"
}
resource "workbench_perimeter" "test" {
  resource_id       = "acme-research"
  owners            = ["admin@example.com"]
  users             = ["user@example.com"]
  sync_google_group = %t
}
`, host, syncGoogleGroup)
}

func testAccPerimeterResourceConfigUpdated(host string) string {
	return fmt.Sprintf(`
provider "workbench" {
  host = "%s"
}
resource "workbench_perimeter" "test" {
  resource_id       = "acme-research"
  owners            = ["admin@example.com", "admin2@example.com"]
  users             = ["user@example.com", "user2@example.com"]
  sync_google_group = true
}
`, host)
}

func testAccPerimeterResourceConfigSynced(host string) string {
	return fmt.Sprintf(`
provider "workbench" {
  host = "%s"
}
resource "workbench_perimeter" "test" {
  resource_id       = "acme-research"
  owners            = ["admin@example.com"]
  users             = ["user@example.com"]
  sync_google_group = true
}
`, host)
}
