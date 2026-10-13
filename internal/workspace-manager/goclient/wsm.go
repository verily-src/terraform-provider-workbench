// package goclient provides utilties for working with the workspace-manager client
package goclient

import (
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/wsm"
)

// WithAuthToken creates a wsm client option that uses the provided token for authentication.
func WithAuthToken(token string) wsm.ClientOption {
	return wsm.WithRequestEditorFn(TokenIntercept(token))
}
