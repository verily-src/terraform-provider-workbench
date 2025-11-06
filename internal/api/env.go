package api

import (
	"context"
	"github.com/verily-src/terraform-provider-workbench/internal/client"
	"net/http"
)

func createHttpClient(ctx context.Context, host string, useIdtoken bool, impersonateServiceAccount string) (*http.Client, error) {
	if impersonateServiceAccount != "" {
		return client.OauthClientWithImpersonation(ctx, impersonateServiceAccount)
	}

	if useIdtoken {
		return client.OauthClientFromWorkloadIdentity(ctx, host)
	}
	return client.OauthClientFromGoogleCredentials(ctx)
}
