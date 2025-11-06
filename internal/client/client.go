// package client provides utilites for working with Workbench clients.
package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"reflect"

	"github.com/deepmap/oapi-codegen/pkg/securityprovider"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken"
	"google.golang.org/api/impersonate"
	"google.golang.org/api/option"
)

// OauthClientFromGoogleCredentials creates a http client for use with Workbench APIs using
// Google application default credentials.
func OauthClientFromGoogleCredentials(ctx context.Context) (*http.Client, error) {
	scopes := []string{
		"https://www.googleapis.com/auth/cloud-platform",
	}
	creds, err := google.FindDefaultCredentials(ctx, scopes...)
	if err != nil {
		return nil, fmt.Errorf("unable to get default credentials: %w", err)
	}

	return oauth2.NewClient(ctx, creds.TokenSource), nil
}

// OauthClientWithImpersonation creates an http client that impersonates the given service account.
// The impersonate library uses application default credentials (ADC) as the base, which automatically
// handles workload identity in GKE environments.
func OauthClientWithImpersonation(ctx context.Context, serviceAccount string) (*http.Client, error) {
	credentials, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, fmt.Errorf("failed to generate default credentials: %v", err)
	}

	ts, err := impersonate.CredentialsTokenSource(ctx, impersonate.CredentialsConfig{
		TargetPrincipal: serviceAccount,
		Scopes:          []string{"openid", "https://www.googleapis.com/auth/userinfo.email"},
	}, option.WithCredentials(credentials))
	if err != nil {
		return nil, fmt.Errorf("unable to create impersonation token source for %s: %w", serviceAccount, err)
	}

	return oauth2.NewClient(ctx, ts), nil
}

// OauthClientFromWorkloadIdentity creates an http client for communication between Workbench services internally inside a GKE cluster
func OauthClientFromWorkloadIdentity(ctx context.Context, audience string) (*http.Client, error) {
	tokenSource, err := idtoken.NewTokenSource(ctx, audience)
	if err != nil {
		return nil, fmt.Errorf("unable to create token source: %w", err)
	}

	return oauth2.NewClient(ctx, tokenSource), nil
}

// TokenIntercept creates an http intercept handler that adds the provided auth token to requests.
func TokenIntercept(token string) func(context.Context, *http.Request) error {
	bearerTokenProvider, err := securityprovider.NewSecurityProviderBearerToken(token)
	if err != nil {
		// NewSecurityProviderBearerToken builds a struct, never returning an error.
		log.Fatalf("Error creating bearer token: %v", err)
	}
	return bearerTokenProvider.Intercept
}

type response interface {
	Status() string
	StatusCode() int
}

func ResponseError[T response](resp T, err error) (T, error) {
	if err != nil {
		// Some kind of transport error where we don't get a response code.
		return resp, err
	}
	if resp.StatusCode() >= 200 && resp.StatusCode() < 300 {
		return resp, nil
	}
	if ae := extractApiError(resp); ae != nil {
		return resp, ae
	}
	if m := resp.Status(); m != "" {
		return resp, errors.New(m)
	}
	return resp, fmt.Errorf("http response: %v", resp.StatusCode())
}

func NewApiError(code int, message string) ApiError {
	return ApiError{StatusCode: code, Message: message}
}

func ApiErrorf(code int, format string, a ...interface{}) ApiError {
	return NewApiError(code, fmt.Sprintf(format, a...))
}

type ApiError struct {
	Causes     []string `json:"causes"`
	Message    string   `json:"message"`
	StatusCode int      `json:"statusCode"`
}

func (e ApiError) Error() string {
	if e.Message == "" {
		if e.StatusCode != 0 {
			return fmt.Sprintf("api response: %v", e.StatusCode)
		}
		return "Unknown error"
	}
	return e.Message
}

func ApiErrorIsCode(err error, code int) bool {
	apiErr, ok := err.(*ApiError)
	if ok {
		return apiErr.StatusCode == code
	}
	return false
}

func extractApiError(resp any) *ApiError {
	val := reflect.ValueOf(resp)
	if val.Kind() == reflect.Ptr && !val.IsNil() {
		// If it's a pointer, get the underlying value.
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil
	}
	bodyField := val.FieldByName("Body")
	if !bodyField.IsValid() || bodyField.Type() != reflect.TypeOf([]byte{}) {
		return nil
	}
	body, ok := bodyField.Interface().([]byte)
	if !ok || len(body) == 0 {
		return nil
	}
	var ae ApiError
	if err := json.Unmarshal(body, &ae); err != nil {
		return nil
	}
	return &ae
}

// Ptr is a utility to turn constants into pointers, commonly used in openapi clients.
func Ptr[T any](s T) *T {
	return &s
}
