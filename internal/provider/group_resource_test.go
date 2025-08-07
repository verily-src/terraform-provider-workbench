package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccGroupResource(t *testing.T) {
	host := setupFakes(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccGroupResourceConfig(host, 30, true, false, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_group.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("This is a test group"),
					),
					statecheck.ExpectKnownValue(
						"workbench_group.test",
						tfjsonpath.New("group_name"),
						knownvalue.StringExact("test-group"),
					),
					statecheck.ExpectKnownValue(
						"workbench_group.test",
						tfjsonpath.New("organization_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_group.test",
						tfjsonpath.New("organization_user_facing_id"),
						knownvalue.StringExact("test-org"),
					),
					statecheck.ExpectKnownValue(
						"workbench_group.test",
						tfjsonpath.New("created_by"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_group.test",
						tfjsonpath.New("created_date"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_group.test",
						tfjsonpath.New("last_updated_by"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_group.test",
						tfjsonpath.New("last_updated_date"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_group.test",
						tfjsonpath.New("expiration_days"),
						knownvalue.Int64Exact(30),
					),
					statecheck.ExpectKnownValue(
						"workbench_group.test",
						tfjsonpath.New("expiration_notification"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"workbench_group.test",
						tfjsonpath.New("sync_group"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"workbench_group.test",
						tfjsonpath.New("require_grant_reason"),
						knownvalue.Bool(false),
					),
				},
			},
			// Update and Read testing
			{
				Config: testAccGroupResourceConfig(host, 60, false, true, true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_group.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("This is a test group"),
					),
					statecheck.ExpectKnownValue(
						"workbench_group.test",
						tfjsonpath.New("expiration_days"),
						knownvalue.Int64Exact(60),
					),
					statecheck.ExpectKnownValue(
						"workbench_group.test",
						tfjsonpath.New("expiration_notification"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"workbench_group.test",
						tfjsonpath.New("sync_group"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"workbench_group.test",
						tfjsonpath.New("require_grant_reason"),
						knownvalue.Bool(true),
					),
				},
			},
			// Deletion is handled automatically by the testing framework.
		},
	})
}

func testAccGroupResourceConfig(host string, expirationDays int, expirationNotification bool, syncGroup bool, requireGrantReason bool) string {
	return fmt.Sprintf(`
provider "workbench" {
  host = "%s"
}
resource "workbench_group" "test" {
  group_name                  = "test-group"
  organization_user_facing_id = "test-org"
  expiration_days             = %d
  expiration_notification     = %t
  sync_group                  = %t
  require_grant_reason        = %t
  description                 = "This is a test group"
}
`, host, expirationDays, expirationNotification, syncGroup, requireGrantReason)
}
