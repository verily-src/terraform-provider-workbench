package api

import (
	"context"
	"github.com/verily-src/terraform-provider-workbench/internal/client"
	"net/http"
	"os"
)

const (
	useIdToken = "USE_ID_TOKEN"
)

func createHttpClient(ctx context.Context, host string) (*http.Client, error) {
	if os.Getenv(useIdToken) == "true" {
		return client.OauthClientFromWorkloadIdentity(ctx, host)
	}
	return client.OauthClientFromGoogleCredentials(ctx)
}
