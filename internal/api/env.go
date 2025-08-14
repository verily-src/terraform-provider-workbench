package api

import (
	"context"
	"github.com/verily-src/terraform-provider-workbench/internal/client"
	"net/http"
)

func createHttpClient(ctx context.Context, host string, useIdtoken bool) (*http.Client, error) {
	if useIdtoken {
		return client.OauthClientFromWorkloadIdentity(ctx, host)
	}
	return client.OauthClientFromGoogleCredentials(ctx)
}
