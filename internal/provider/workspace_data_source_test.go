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

func TestAccWorkspaceDataSource(t *testing.T) {
	ID := uuid.New()
	orgID := uuid.New()
	podID := uuid.New()
	now := time.Now()
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == fmt.Sprintf("/api/wsm/api/workspaces/v1/%s", ID.String()) {
			desc := &wsm.WorkspaceDescription{
				Id:              ID,
				UserFacingId:    "test-workspace",
				Description:     client.Ptr("This is a test workspace"),
				DisplayName:     client.Ptr("Test Workspace"),
				OrgId:           &orgID,
				CrgId:           &podID,
				CreatedBy:       "test-user",
				CreatedDate:     now,
				LastUpdatedBy:   "test-user",
				LastUpdatedDate: now,
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
				Config: fmt.Sprintf(testAccWorkspaceDataSourceConfig, mockServer.URL, ID.String()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.workbench_workspace.test", "id", ID.String()),
					resource.TestCheckResourceAttr("data.workbench_workspace.test", "user_facing_id", "test-workspace"),
					resource.TestCheckResourceAttr("data.workbench_workspace.test", "description", "This is a test workspace"),
					resource.TestCheckResourceAttr("data.workbench_workspace.test", "display_name", "Test Workspace"),
					resource.TestCheckResourceAttr("data.workbench_workspace.test", "organization_id", orgID.String()),
					resource.TestCheckResourceAttr("data.workbench_workspace.test", "pod_id", podID.String()),
					resource.TestCheckResourceAttr("data.workbench_workspace.test", "created_by", "test-user"),
					resource.TestCheckResourceAttr("data.workbench_workspace.test", "created_date", now.Format(time.RFC3339)),
					resource.TestCheckResourceAttr("data.workbench_workspace.test", "last_updated_by", "test-user"),
					resource.TestCheckResourceAttr("data.workbench_workspace.test", "last_updated_date", now.Format(time.RFC3339)),
				),
			},
		},
	})
}

const testAccWorkspaceDataSourceConfig = `
provider "workbench" {
  host = "%s"
}
data "workbench_workspace" "test" {
  id = "%s"
}
`
