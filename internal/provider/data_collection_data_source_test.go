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

func TestAccDataCollectionDataSource(t *testing.T) {
	ID := uuid.New()
	orgID := uuid.New()
	podID := uuid.New()
	now := time.Now()
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == fmt.Sprintf("/api/wsm/api/workspaces/v1/%s", ID.String()) {
			desc := &wsm.WorkspaceDescription{
				Id:              ID,
				UserFacingId:    "test-data-collection",
				Description:     client.Ptr("This is a test data collection"),
				DisplayName:     client.Ptr("Test Data Collection"),
				OrgId:           &orgID,
				CrgId:           &podID,
				Properties:      buildProperties(map[string]string{"terra-type": "data-collection", "terra-support-email": "testing@example.com", "terra-organization-name": "verily-test-org", "terra-therapeutic-tags": "[\"cardiology\", \"dermatology\"]", "terra-update-frequency": "weekly"}),
				CreatedBy:       "test-user",
				CreatedDate:     now,
				LastUpdatedBy:   "test-user",
				LastUpdatedDate: now,
				GcpContext: &wsm.GcpContext{
					ProjectId: "test-data-collection-project",
				},
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(desc)
		}
	}))
	defer mockServer.Close()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: fmt.Sprintf(testAccDataCollectionDataSourceConfig, mockServer.URL, ID.String()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.workbench_data_collection.test", "id", ID.String()),
					resource.TestCheckResourceAttr("data.workbench_data_collection.test", "user_facing_id", "test-data-collection"),
					resource.TestCheckResourceAttr("data.workbench_data_collection.test", "description", "This is a test data collection"),
					resource.TestCheckResourceAttr("data.workbench_data_collection.test", "display_name", "Test Data Collection"),
					resource.TestCheckResourceAttr("data.workbench_data_collection.test", "organization_id", orgID.String()),
					resource.TestCheckResourceAttr("data.workbench_data_collection.test", "pod_id", podID.String()),
					resource.TestCheckResourceAttr("data.workbench_data_collection.test", "created_by", "test-user"),
					resource.TestCheckResourceAttr("data.workbench_data_collection.test", "created_date", now.Format(time.RFC3339)),
					resource.TestCheckResourceAttr("data.workbench_data_collection.test", "last_updated_by", "test-user"),
					resource.TestCheckResourceAttr("data.workbench_data_collection.test", "last_updated_date", now.Format(time.RFC3339)),
					resource.TestCheckResourceAttr("data.workbench_data_collection.test", "support_email", "testing@example.com"),
					resource.TestCheckResourceAttr("data.workbench_data_collection.test", "organization_name", "verily-test-org"),
					resource.TestCheckTypeSetElemAttr("data.workbench_data_collection.test", "therapeutic_tags.*", "cardiology"),
					resource.TestCheckTypeSetElemAttr("data.workbench_data_collection.test", "therapeutic_tags.*", "dermatology"),
					resource.TestCheckResourceAttr("data.workbench_data_collection.test", "update_frequency", "weekly"),
					resource.TestCheckResourceAttr("data.workbench_data_collection.test", "gcp_project_id", "test-data-collection-project"),
					resource.TestCheckNoResourceAttr("data.workbench_data_collection.test", "aws_account_id"),
				),
			},
		},
	})
}

const testAccDataCollectionDataSourceConfig = `
provider "workbench" {
  host = "%s"
}
data "workbench_data_collection" "test" {
  id = "%s"
}
`
