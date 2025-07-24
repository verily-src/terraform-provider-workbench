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

func TestAccControlledGcsBucketResource(t *testing.T) {
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
				Config: testAccControlledGcsBucketResourceConfig(host, ws1, ws2, "ws1", "test-location", "STANDARD", "COPY_NOTHING"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_controlled_gcs_bucket.test",
						tfjsonpath.New("workspace_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_controlled_gcs_bucket.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_controlled_gcs_bucket.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("test-bucket"),
					),
					statecheck.ExpectKnownValue(
						"workbench_controlled_gcs_bucket.test",
						tfjsonpath.New("bucket_name"),
						knownvalue.StringExact("test-bucket-1234"),
					),
					statecheck.ExpectKnownValue(
						"workbench_controlled_gcs_bucket.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("A test controlled GCS bucket"),
					),
					statecheck.ExpectKnownValue(
						"workbench_controlled_gcs_bucket.test",
						tfjsonpath.New("location"),
						knownvalue.StringExact("test-location"),
					),
					statecheck.ExpectKnownValue(
						"workbench_controlled_gcs_bucket.test",
						tfjsonpath.New("storage_class"),
						knownvalue.StringExact("STANDARD"),
					),
					statecheck.ExpectKnownValue(
						"workbench_controlled_gcs_bucket.test",
						tfjsonpath.New("clone_instruction"),
						knownvalue.StringExact("COPY_NOTHING"),
					),
				},
			},
			// Update and Read testing
			{
				Config: testAccControlledGcsBucketResourceConfig(host, ws1, ws2, "ws1", "test-location", "COLDLINE", "COPY_RESOURCE"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_controlled_gcs_bucket.test",
						tfjsonpath.New("workspace_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_controlled_gcs_bucket.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("test-bucket"),
					),
					statecheck.ExpectKnownValue(
						"workbench_controlled_gcs_bucket.test",
						tfjsonpath.New("bucket_name"),
						knownvalue.StringExact("test-bucket-1234"),
					),
					statecheck.ExpectKnownValue(
						"workbench_controlled_gcs_bucket.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("A test controlled GCS bucket"),
					),
					statecheck.ExpectKnownValue(
						"workbench_controlled_gcs_bucket.test",
						tfjsonpath.New("location"),
						knownvalue.StringExact("test-location"),
					),
					statecheck.ExpectKnownValue(
						"workbench_controlled_gcs_bucket.test",
						tfjsonpath.New("storage_class"),
						knownvalue.StringExact("COLDLINE"),
					),
					statecheck.ExpectKnownValue(
						"workbench_controlled_gcs_bucket.test",
						tfjsonpath.New("clone_instruction"),
						knownvalue.StringExact("COPY_RESOURCE"),
					),
				},
			},
			// Changing workspace ID should cause recreate
			{
				Config: testAccControlledGcsBucketResourceConfig(host, ws1, ws2, "ws2", "test-location", "COLDLINE", "COPY_RESOURCE"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"workbench_controlled_gcs_bucket.test",
						tfjsonpath.New("workspace_id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"workbench_controlled_gcs_bucket.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("test-bucket"),
					),
					statecheck.ExpectKnownValue(
						"workbench_controlled_gcs_bucket.test",
						tfjsonpath.New("location"),
						knownvalue.StringExact("test-location"),
					),
					statecheck.ExpectKnownValue(
						"workbench_controlled_gcs_bucket.test",
						tfjsonpath.New("storage_class"),
						knownvalue.StringExact("COLDLINE"),
					),
					statecheck.ExpectKnownValue(
						"workbench_controlled_gcs_bucket.test",
						tfjsonpath.New("clone_instruction"),
						knownvalue.StringExact("COPY_RESOURCE"),
					),
				},
			},
			// Import testing
			{
				ImportState: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					workspaceID, bucketID, err := getWorkspaceAndBucketID(s, "test")
					if err != nil {
						return "", err
					}
					return fmt.Sprintf("workspaces/%s/controlled_gcs_buckets/%s", workspaceID, bucketID), nil
				},
				ResourceName:                         "workbench_controlled_gcs_bucket.test",
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "id",
				ImportStateVerifyIgnore: []string{
					"storage_class", // storage_class is not part of the GET response.
				},
			},
			// Deletion is handled automatically by the testing framework.
		},
	})
}

func getWorkspaceAndBucketID(s *terraform.State, bucketTFID string) (string, string, error) {
	rs, ok := s.RootModule().Resources[fmt.Sprintf("workbench_controlled_gcs_bucket.%s", bucketTFID)]
	if !ok {
		return "", "", fmt.Errorf("controlled GCS bucket %s not found in state", bucketTFID)
	}
	bucketID, ok := rs.Primary.Attributes["id"]
	if !ok {
		return "", "", fmt.Errorf("ID for controlled GCS bucket %s not found in state", bucketTFID)
	}
	workspaceID, ok := rs.Primary.Attributes["workspace_id"]
	if !ok {
		return "", "", fmt.Errorf("workspace ID for controlled GCS bucket %s not found in state", bucketTFID)
	}
	return workspaceID, bucketID, nil
}

func testAccControlledGcsBucketResourceConfig(host string, workspace1, workspace2 configModifier, bucketWorkspace, location, storageClass, cloneInstruction string) string {
	return generateConfig(
		withProvider(host),
		workspace1,
		workspace2,
		withRaw(fmt.Sprintf(`
resource "workbench_controlled_gcs_bucket" "test" {
  workspace_id = %s
  name = "test-bucket"
  bucket_name = "test-bucket-1234"
  description = "A test controlled GCS bucket"
  location = %q
  storage_class = %q
  clone_instruction = %q
}
`, workspaceByReference(bucketWorkspace), location, storageClass, cloneInstruction)))
}
