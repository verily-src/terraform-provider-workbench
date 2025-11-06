package provider

import (
	"testing"

	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/verily-src/terraform-provider-workbench/internal/client"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/wsm"
)

func TestAccWorkspaceIamBindingDataSource(t *testing.T) {
	workspaceID := uuid.New()
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == fmt.Sprintf("/api/wsm/api/workspaces/v1/%s/roles", workspaceID.String()) {
			desc := &wsm.RoleBindingList{
				{
					Members: client.Ptr([]string{"test-owner"}),
					Role:    wsm.IamRoleOWNER,
				},
				{
					Members: client.Ptr([]string{"test-user1", "test-user2"}),
					Role:    wsm.IamRoleWRITER,
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
				Config: testAccWorkspaceIamBindingDataSourceConfig(mockServer.URL, workspaceID.String()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.workbench_workspace_iam_binding.test-owner", "workspace_id", workspaceID.String()),
					resource.TestCheckResourceAttr("data.workbench_workspace_iam_binding.test-owner", "role", "OWNER"),
					resource.TestCheckResourceAttr("data.workbench_workspace_iam_binding.test-owner", "members.#", "1"),
					resource.TestCheckResourceAttr("data.workbench_workspace_iam_binding.test-owner", "members.0", "test-owner"),

					resource.TestCheckResourceAttr("data.workbench_workspace_iam_binding.test-writer", "workspace_id", workspaceID.String()),
					resource.TestCheckResourceAttr("data.workbench_workspace_iam_binding.test-writer", "role", "WRITER"),
					resource.TestCheckResourceAttr("data.workbench_workspace_iam_binding.test-writer", "members.#", "2"),
					resource.TestCheckResourceAttr("data.workbench_workspace_iam_binding.test-writer", "members.0", "test-user1"),
					resource.TestCheckResourceAttr("data.workbench_workspace_iam_binding.test-writer", "members.1", "test-user2"),

					resource.TestCheckResourceAttr("data.workbench_workspace_iam_binding.test-reader", "workspace_id", workspaceID.String()),
					resource.TestCheckResourceAttr("data.workbench_workspace_iam_binding.test-reader", "role", "READER"),
					resource.TestCheckResourceAttr("data.workbench_workspace_iam_binding.test-reader", "members.#", "0"),
				),
			},
		},
	})
}

func testAccWorkspaceIamBindingDataSourceConfig(url, workspaceID string) string {
	return generateConfig(
		withProvider(url),
		withRaw(fmt.Sprintf(`
data "workbench_workspace_iam_binding" "test-owner" {
  workspace_id = %q
  role      = "OWNER"
}
data "workbench_workspace_iam_binding" "test-writer" {
  workspace_id = %q
  role      = "WRITER"
}
data "workbench_workspace_iam_binding" "test-reader" {
  workspace_id = %q
  role      = "READER"
}
`, workspaceID, workspaceID, workspaceID)),
	)
}
