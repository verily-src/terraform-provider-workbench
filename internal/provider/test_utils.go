package provider

import (
	"fmt"
	"strings"
	"testing"

	"github.com/verily-src/terraform-provider-workbench/internal/fakes"
	samfake "github.com/verily-src/terraform-provider-workbench/internal/fakes/sam"
	userfake "github.com/verily-src/terraform-provider-workbench/internal/fakes/user"
	wsmfake "github.com/verily-src/terraform-provider-workbench/internal/fakes/wsm"
)

type configBuilder struct {
	strings.Builder
}

func (cb *configBuilder) WriteLine(indent int, line string) {
	indentation := strings.Repeat("  ", indent)
	cb.WriteString(fmt.Sprintf("%s%s\n", indentation, line))
}

type configModifier func(cb *configBuilder)

func generateConfig(resources ...configModifier) string {
	var cb configBuilder
	for _, resource := range resources {
		resource(&cb)
	}
	return cb.String()
}

func withRaw(raw string) configModifier {
	return func(cb *configBuilder) {
		cb.WriteString(raw)
	}
}

func withProvider(host string) configModifier {
	return withRaw(fmt.Sprintf(`
provider "workbench" {
  host = %q
}
`, host))
}

func withWorkspace(id, userFacingID, orgID, podID string) configModifier {
	return withRaw(fmt.Sprintf(`
resource "workbench_workspace" %q {
  user_facing_id = %q
  organization_id = %q
  pod_id = %q
}`, id, userFacingID, orgID, podID))
}

func withGroup(id, groupName, orgUfid string) configModifier {
	return withRaw(fmt.Sprintf(`
resource "workbench_group" %q {
  group_name = %q
  organization_user_facing_id = %q
}`, id, groupName, orgUfid))
}

func workspaceByReference(terraformID string) string {
	return fmt.Sprintf("workbench_workspace.%s.id", terraformID)
}

func groupNameByReference(terraformID string) string {
	return fmt.Sprintf("workbench_group.%s.group_name", terraformID)
}

func groupOrgByReference(terraformID string) string {
	return fmt.Sprintf("workbench_group.%s.organization_id", terraformID)
}

func setupFakes(t *testing.T) string {
	addr := fakes.Use(t,
		wsmfake.New(),
		userfake.New(),
		samfake.New())

	return fmt.Sprintf("http://%s", addr)
}
