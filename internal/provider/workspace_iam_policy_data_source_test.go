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

func TestAccWorkspaceIamPolicyDataSource(t *testing.T) {
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
				{
					Members: client.Ptr([]string{"test-user1", "test-user3"}),
					Role:    wsm.IamRoleREADER,
				},
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(desc)
		}
	}))
	defer mockServer.Close()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: fmt.Sprintf(testAccWorkspaceIamPolicyDataSourceConfig, mockServer.URL, workspaceID.String()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.workbench_workspace_iam_policy.test", "workspace_id", workspaceID.String()),
					resource.TestCheckResourceAttr("data.workbench_workspace_iam_policy.test", "iams.#", "3"),

					resource.TestCheckResourceAttr("data.workbench_workspace_iam_policy.test", "iams.0.role", "OWNER"),
					resource.TestCheckResourceAttr("data.workbench_workspace_iam_policy.test", "iams.0.members.#", "1"),
					resource.TestCheckResourceAttr("data.workbench_workspace_iam_policy.test", "iams.0.members.0", "test-owner"),

					resource.TestCheckResourceAttr("data.workbench_workspace_iam_policy.test", "iams.1.role", "WRITER"),
					resource.TestCheckResourceAttr("data.workbench_workspace_iam_policy.test", "iams.1.members.#", "2"),
					resource.TestCheckResourceAttr("data.workbench_workspace_iam_policy.test", "iams.1.members.0", "test-user1"),
					resource.TestCheckResourceAttr("data.workbench_workspace_iam_policy.test", "iams.1.members.1", "test-user2"),

					resource.TestCheckResourceAttr("data.workbench_workspace_iam_policy.test", "iams.2.role", "READER"),
					resource.TestCheckResourceAttr("data.workbench_workspace_iam_policy.test", "iams.2.members.#", "2"),
					resource.TestCheckResourceAttr("data.workbench_workspace_iam_policy.test", "iams.2.members.0", "test-user1"),
					resource.TestCheckResourceAttr("data.workbench_workspace_iam_policy.test", "iams.2.members.1", "test-user3"),
				),
			},
		},
	})
}

const testAccWorkspaceIamPolicyDataSourceConfig = `
provider "workbench" {
  host = "%s"
}
data "workbench_workspace_iam_policy" "test" {
  workspace_id = "%s"
}
`
