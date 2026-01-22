package provider

import (
	"regexp"
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
				GcpContext: &wsm.GcpContext{
					ProjectId: "test-gcp-project-123",
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
					resource.TestCheckResourceAttr("data.workbench_workspace.test", "gcp_project_id", "test-gcp-project-123"),
					resource.TestCheckNoResourceAttr("data.workbench_workspace.test", "aws_account_id"),
				),
			},
		},
	})
}

func TestAccWorkspaceDataSourceByUserFacingId(t *testing.T) {
	ID := uuid.New()
	orgID := uuid.New()
	podID := uuid.New()
	now := time.Now()
	userFacingId := "test-workspace-ufid"
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == fmt.Sprintf("/api/wsm/api/workspaces/v1/workspaceByUserFacingId/%s", userFacingId) {
			desc := &wsm.WorkspaceDescription{
				Id:              ID,
				UserFacingId:    userFacingId,
				Description:     client.Ptr("This is a test workspace by user facing id"),
				DisplayName:     client.Ptr("Test Workspace Ufid"),
				OrgId:           &orgID,
				CrgId:           &podID,
				CreatedBy:       "test-user",
				CreatedDate:     now,
				LastUpdatedBy:   "test-user",
				LastUpdatedDate: now,
				GcpContext: &wsm.GcpContext{
					ProjectId: "test-gcp-project-ufid-123",
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
			// Read testing by user_facing_id
			{
				Config: fmt.Sprintf(testAccWorkspaceDataSourceByUserFacingIdConfig, mockServer.URL, userFacingId),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.workbench_workspace.test_ufid", "id", ID.String()),
					resource.TestCheckResourceAttr("data.workbench_workspace.test_ufid", "user_facing_id", userFacingId),
					resource.TestCheckResourceAttr("data.workbench_workspace.test_ufid", "description", "This is a test workspace by user facing id"),
					resource.TestCheckResourceAttr("data.workbench_workspace.test_ufid", "display_name", "Test Workspace Ufid"),
					resource.TestCheckResourceAttr("data.workbench_workspace.test_ufid", "organization_id", orgID.String()),
					resource.TestCheckResourceAttr("data.workbench_workspace.test_ufid", "pod_id", podID.String()),
					resource.TestCheckResourceAttr("data.workbench_workspace.test_ufid", "created_by", "test-user"),
					resource.TestCheckResourceAttr("data.workbench_workspace.test_ufid", "created_date", now.Format(time.RFC3339)),
					resource.TestCheckResourceAttr("data.workbench_workspace.test_ufid", "last_updated_by", "test-user"),
					resource.TestCheckResourceAttr("data.workbench_workspace.test_ufid", "last_updated_date", now.Format(time.RFC3339)),
					resource.TestCheckResourceAttr("data.workbench_workspace.test_ufid", "gcp_project_id", "test-gcp-project-ufid-123"),
					resource.TestCheckNoResourceAttr("data.workbench_workspace.test_ufid", "aws_account_id"),
				),
			},
		},
	})
}

func TestAccWorkspaceDataSourceMissingIdentifier(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This should never be called since the validation should fail first
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test that error is returned when neither id nor user_facing_id is provided
			{
				Config:      fmt.Sprintf(testAccWorkspaceDataSourceMissingIdentifierConfig, mockServer.URL),
				ExpectError: regexp.MustCompile("Either 'id' or 'user_facing_id' must be provided"),
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

const testAccWorkspaceDataSourceByUserFacingIdConfig = `
provider "workbench" {
  host = "%s"
}
data "workbench_workspace" "test_ufid" {
  user_facing_id = "%s"
}
`

const testAccWorkspaceDataSourceMissingIdentifierConfig = `
provider "workbench" {
  host = "%s"
}
data "workbench_workspace" "test" {
}
`
