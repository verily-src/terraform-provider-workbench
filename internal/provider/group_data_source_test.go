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
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/user"
)

func TestAccGroupDataSource(t *testing.T) {
	groupName := "test-group"
	orgID := uuid.New()
	orgUfid := "test-org-ufid"
	now := time.Now()
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == fmt.Sprintf("/api/user/api/groups/v1/%s", groupName) {
			desc := &user.GroupDescriptionAndRoles{
				GroupDescription: &user.GroupDescription{
					GroupEmail:             "test-group@gmail.com",
					GroupName:              client.Ptr("test-group"),
					InternalName:           client.Ptr("test-group-internal"),
					OrgId:                  orgID.String(),
					OrgUfid:                orgUfid,
					CreatedBy:              "test-user",
					CreatedDate:            now,
					LastUpdatedBy:          "test-user",
					LastUpdatedDate:        now,
					ExpirationDays:         30,
					ExpirationNotification: client.Ptr(true),
					RequireGrantReason:     client.Ptr(true),
					SyncGroup:              true,
					Description:            client.Ptr("This is a test group"),
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
				Config: fmt.Sprintf(`
					provider "workbench" {
  						host = "%s"
					}
					data "workbench_group" "test" {
  						group_name = "%s"
  						organization_user_facing_id   = "%s"
					}
				`, mockServer.URL, groupName, orgUfid),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.workbench_group.test", "group_name", groupName),
					resource.TestCheckResourceAttr("data.workbench_group.test", "internal_name", "test-group-internal"),
					resource.TestCheckResourceAttr("data.workbench_group.test", "description", "This is a test group"),
					resource.TestCheckResourceAttr("data.workbench_group.test", "organization_user_facing_id", orgUfid),
					resource.TestCheckResourceAttr("data.workbench_group.test", "organization_id", orgID.String()),
					resource.TestCheckResourceAttr("data.workbench_group.test", "group_email", "test-group@gmail.com"),
					resource.TestCheckResourceAttr("data.workbench_group.test", "expiration_days", "30"),
					resource.TestCheckResourceAttr("data.workbench_group.test", "expiration_notification", "true"),
					resource.TestCheckResourceAttr("data.workbench_group.test", "require_grant_reason", "true"),
					resource.TestCheckResourceAttr("data.workbench_group.test", "sync_group", "true"),
					resource.TestCheckResourceAttr("data.workbench_group.test", "created_by", "test-user"),
					resource.TestCheckResourceAttr("data.workbench_group.test", "created_date", now.Format(time.RFC3339)),
					resource.TestCheckResourceAttr("data.workbench_group.test", "last_updated_by", "test-user"),
					resource.TestCheckResourceAttr("data.workbench_group.test", "last_updated_date", now.Format(time.RFC3339)),
				),
			},
		},
	})
}

func TestAccWorkspaceDataSource_noorg(t *testing.T) {
	groupName := "test-group"

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == fmt.Sprintf("/api/user/api/groups/v1/%s", groupName) {
			desc := &user.GroupDescriptionAndRoles{
				GroupAndRoles: user.GroupAndRoles{
					GroupEmail: "test-group@gmail.com",
					GroupName:  groupName,
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
				Config: fmt.Sprintf(`
					provider "workbench" {
  						host = "%s"
					}
					data "workbench_group" "test" {
  						group_name = "%s"
					}
  				`, mockServer.URL, groupName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.workbench_group.test", "group_name", groupName),
					resource.TestCheckResourceAttr("data.workbench_group.test", "group_email", "test-group@gmail.com"),
				),
			},
		},
	})
}

const testAccGroupDataSourceConfig = `
provider "workbench" {
  host = "%s"
}
data "workbench_group" "test" {
  group_name = "%s"
  organization_user_facing_id   = "%s"
}
`
