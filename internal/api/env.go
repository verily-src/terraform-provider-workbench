package api

import (
	"context"
	"net/http"

	wsmClient "github.com/verily-src/terraform-provider-workbench/internal/workspace-manager/goclient"
)

func createHttpClient(ctx context.Context, host string, useIdtoken bool, useCLICredentials bool, impersonateServiceAccount string) (*http.Client, error) {
	if impersonateServiceAccount != "" {
		return wsmClient.OauthClientWithImpersonation(ctx, impersonateServiceAccount)
	}

	if useCLICredentials {
		return wsmClient.OauthClientFromCLI(ctx)
	}

	if useIdtoken {
		return wsmClient.OauthClientFromWorkloadIdentity(ctx, host)
	}
	return wsmClient.OauthClientFromGoogleCredentials(ctx)
}
