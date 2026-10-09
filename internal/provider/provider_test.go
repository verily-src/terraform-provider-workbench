package provider

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/user"
	wsmclient "github.com/verily-src/terraform-provider-workbench/internal/workspace-manager/goclient"
)

var (
	// testAccProtoV6ProviderFactories are used to instantiate a provider during
	// acceptance testing. The factory function will be invoked for every Terraform
	// CLI command executed to create a provider server to which the CLI can
	// reattach.
	testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"workbench": providerserver.NewProtocol6WithError(New("test")()),
	}
)

// TestAccProviderUsesCLICredentials is an end-to-end check that
// `use_cli_credentials = true` wires all the way through the provider: the
// schema attribute is read in Configure, createHttpClient selects the CLI
// branch, and the token returned by `wb auth print-access-token` is attached as
// a Bearer token on the outgoing request. A stub `wb` binary on PATH stands in
// for the real CLI, and the mock server captures the Authorization header it
// receives so we can assert the CLI-provided token actually reached the wire.
func TestAccProviderUsesCLICredentials(t *testing.T) {
	groupName := "test-group"
	orgID := uuid.New()
	orgUfid := "test-org-ufid"
	now := time.Now()

	// The token the stub CLI will hand back. Only the payload segment is parsed
	// (for its exp claim), so the header/signature segments are placeholders.
	wantToken := makeStubJWT(t, now.Add(time.Hour))
	stubWorkbenchCLI(t, wantToken)

	var mu sync.Mutex
	var gotAuthHeader string
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == fmt.Sprintf("/api/user/api/groups/v1/%s", groupName) {
			mu.Lock()
			gotAuthHeader = r.Header.Get("Authorization")
			mu.Unlock()

			desc := &user.GroupDescriptionAndRoles{
				GroupDescription: &user.GroupDescription{
					GroupEmail: "test-group@gmail.com",
					GroupName:  wsmclient.Ptr(groupName),
					OrgId:      orgID.String(),
					OrgUfid:    orgUfid,
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
			{
				Config: fmt.Sprintf(`
provider "workbench" {
  host                = %q
  use_cli_credentials = true
}
data "workbench_group" "test" {
  group_name                  = %q
  organization_user_facing_id = %q
}
`, mockServer.URL, groupName, orgUfid),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.workbench_group.test", "group_name", groupName),
					resource.TestCheckResourceAttr("data.workbench_group.test", "organization_user_facing_id", orgUfid),
				),
			},
		},
	})

	// resource.Test is synchronous, so by the time it returns the data source
	// read has completed and gotAuthHeader has been populated.
	mu.Lock()
	defer mu.Unlock()
	if wantAuthHeader := "Bearer " + wantToken; gotAuthHeader != wantAuthHeader {
		t.Errorf("Authorization header = %q, want %q (CLI token was not attached to the request)", gotAuthHeader, wantAuthHeader)
	}
}

// makeStubJWT builds an unsigned JWT whose payload carries the given expiry.
// The provider only base64url-decodes the payload segment to read the exp
// claim, so the header and signature segments are arbitrary placeholders.
func makeStubJWT(t *testing.T, expiry time.Time) string {
	t.Helper()
	payload, err := json.Marshal(map[string]int64{"exp": expiry.Unix()})
	if err != nil {
		t.Fatalf("marshaling stub JWT payload: %v", err)
	}
	enc := base64.RawURLEncoding.EncodeToString
	return strings.Join([]string{
		enc([]byte(`{"alg":"none","typ":"JWT"}`)),
		enc(payload),
		enc([]byte("signature")),
	}, ".")
}

// stubWorkbenchCLI writes a fake `wb` executable into a temp dir and prepends that dir
// to PATH, so the provider's `wb auth print-access-token` invocation resolves to
// the stub and prints the given token. t.Setenv restores PATH when the test ends.
func stubWorkbenchCLI(t *testing.T, token string) {
	t.Helper()
	dir := t.TempDir()
	// JWTs are base64url + '.', so they contain no shell-special characters and
	// are safe to embed in single quotes.
	script := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' '%s'\n", token)
	stubPath := filepath.Join(dir, "wb")
	if err := os.WriteFile(stubPath, []byte(script), 0o755); err != nil {
		t.Fatalf("writing stub wb executable: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}
