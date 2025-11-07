package client

import (
	"context"
	"testing"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func TestOauthClientWithImpersonation(t *testing.T) {
	ctx := context.Background()

	// Check if we have any credentials available (either GOOGLE_APPLICATION_CREDENTIALS or ADC)
	if _, err := google.FindDefaultCredentials(ctx); err != nil {
		t.Skipf("Skipping integration test: no credentials available (try 'gcloud auth application-default login'): %v", err)
	}

	serviceAccount := "ui_testuser_01@test.verily-bvdp.com"

	t.Run("successful impersonation", func(t *testing.T) {
		client, err := OauthClientWithImpersonation(ctx, serviceAccount)
		if err != nil {
			t.Fatalf("OauthClientWithImpersonation failed: %v", err)
		}

		if client == nil {
			t.Fatal("Expected non-nil client, got nil")
		}

		// Verify the client has a transport configured
		if client.Transport == nil {
			t.Error("Expected client to have a Transport configured")
		}

		// Verify the transport is an oauth2 transport
		transport, ok := client.Transport.(*oauth2.Transport)
		if !ok {
			t.Errorf("Expected Transport to be *oauth2.Transport, got %T", client.Transport)
		}
		// Verify the token source is configured
		if transport.Source == nil {
			t.Error("Expected non-nil token source")
		}
	})

	t.Run("empty service account creates client", func(t *testing.T) {
		// Note: The function doesn't validate the service account at creation time.
		// Validation happens when the token is actually requested.
		client, err := OauthClientWithImpersonation(ctx, "")
		if err != nil {
			t.Logf("Got expected error for empty service account: %v", err)
		} else if client != nil {
			t.Log("Client created successfully (validation will happen at token request time)")
		}
	})
}
