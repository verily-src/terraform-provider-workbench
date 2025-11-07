package provider

import (
	"testing"

	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/verily-src/terraform-provider-workbench/internal/client"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/wsm"
)

func TestAccControlledGCSBucketDataSource(t *testing.T) {
	workspaceID := uuid.New()
	rID := uuid.New()
	folderId := uuid.New()
	now := time.Now()
	rLineage := []wsm.ResourceLineageEntry{
		{
			SourceResourceId:  uuid.New(),
			SourceWorkspaceId: uuid.New(),
		},
		{
			SourceResourceId:  uuid.New(),
			SourceWorkspaceId: uuid.New(),
		},
	}
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == fmt.Sprintf("/api/wsm/api/workspaces/v1/%s/resources/controlled/gcp/buckets/%s", workspaceID.String(), rID.String()) {
			desc := &wsm.GcpGcsBucketResource{
				Metadata: wsm.ResourceMetadata{
					ResourceId:          rID,
					Name:                "test-controlled-gcs-bucket",
					DisplayName:         client.Ptr("Test Controlled GCS Bucket"),
					Description:         client.Ptr("This is a test controlled GCS bucket"),
					CreatedBy:           "user@example.com",
					CreatedDate:         now,
					LastUpdatedBy:       "user@example.com",
					LastUpdatedDate:     now,
					StewardshipType:     wsm.CONTROLLED,
					ResourceType:        wsm.GCSBUCKET,
					Properties:          client.Ptr([]wsm.Property{{Key: "key1", Value: "value1"}}),
					ResourceLineage:     client.Ptr(rLineage),
					FolderId:            client.Ptr(folderId),
					CloningInstructions: client.Ptr(wsm.COPYNOTHING),
				},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(desc)
		}
	}))
	defer mockServer.Close()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() {},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccControlledGCSBucketDataSourceConfig, mockServer.URL, rID.String(), workspaceID.String()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.workbench_controlled_gcs_bucket.test", "id", rID.String()),
					resource.TestCheckResourceAttr("data.workbench_controlled_gcs_bucket.test", "display_name", "Test Controlled GCS Bucket"),
					resource.TestCheckResourceAttr("data.workbench_controlled_gcs_bucket.test", "description", "This is a test controlled GCS bucket"),
					resource.TestCheckResourceAttr("data.workbench_controlled_gcs_bucket.test", "properties.0.key", "key1"),
					resource.TestCheckResourceAttr("data.workbench_controlled_gcs_bucket.test", "properties.0.value", "value1"),
					resource.TestCheckResourceAttr("data.workbench_controlled_gcs_bucket.test", "last_updated_date", now.Format(time.RFC3339)),
					resource.TestCheckResourceAttr("data.workbench_controlled_gcs_bucket.test", "last_updated_by", "user@example.com"),
					resource.TestCheckResourceAttr("data.workbench_controlled_gcs_bucket.test", "created_date", now.Format(time.RFC3339)),
					resource.TestCheckResourceAttr("data.workbench_controlled_gcs_bucket.test", "created_by", "user@example.com"),
					resource.TestCheckResourceAttr("data.workbench_controlled_gcs_bucket.test", "folder_id", folderId.String()),
					resource.TestCheckResourceAttr("data.workbench_controlled_gcs_bucket.test", "resource_type", string(wsm.GCSBUCKET)),
					resource.TestCheckResourceAttr("data.workbench_controlled_gcs_bucket.test", "stewardship_type", string(wsm.CONTROLLED)),
					resource.TestCheckResourceAttr("data.workbench_controlled_gcs_bucket.test", "clone_instruction", string(wsm.COPYNOTHING)),
					resource.TestCheckResourceAttr("data.workbench_controlled_gcs_bucket.test", "resource_lineage.#", "2"),
					resource.TestCheckResourceAttr("data.workbench_controlled_gcs_bucket.test", "resource_lineage.0.source_resource_id", rLineage[0].SourceResourceId.String()),
					resource.TestCheckResourceAttr("data.workbench_controlled_gcs_bucket.test", "resource_lineage.0.source_workspace_id", rLineage[0].SourceWorkspaceId.String()),
					resource.TestCheckResourceAttr("data.workbench_controlled_gcs_bucket.test", "resource_lineage.1.source_resource_id", rLineage[1].SourceResourceId.String()),
					resource.TestCheckResourceAttr("data.workbench_controlled_gcs_bucket.test", "resource_lineage.1.source_workspace_id", rLineage[1].SourceWorkspaceId.String()),
				),
			},
		},
	})
}

const testAccControlledGCSBucketDataSourceConfig = `
provider "workbench" {
  host = "%s"
}
data "workbench_controlled_gcs_bucket" "test" {
  id = "%s"
  workspace_id = "%s"
}
`
