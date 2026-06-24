package api

import (
	"context"
	"fmt"

	"github.com/verily-src/terraform-provider-workbench/internal/client"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/sam"
)

func CreatePerimeter(ctx context.Context, cl *sam.ClientWithResponses, resourceId string, owners []string, users []string) error {
	req := sam.CreatePerimeterJSONRequestBody{
		ResourceId: resourceId,
		Policies: map[string]sam.AccessPolicyMembershipRequest{
			"owner": {
				MemberEmails: owners,
				Roles:        []string{"owner"},
				Actions:      []string{},
			},
			"user": {
				MemberEmails: users,
				Roles:        []string{"user"},
				Actions:      []string{},
			},
		},
	}
	rsp, err := client.ResponseError(cl.CreatePerimeterWithResponse(ctx, req))
	if err != nil {
		if rsp != nil && rsp.JSON409 != nil {
			return fmt.Errorf("perimeter %q already exists in Sam; to manage it with Terraform, run: terraform import workbench_perimeter.<name> %s", resourceId, resourceId)
		}
		return fmt.Errorf("creating perimeter: %w", err)
	}
	return nil
}

func GetPerimeterPolicies(ctx context.Context, cl *sam.ClientWithResponses, resourceId string) ([]sam.AccessPolicyResponseEntryV2, error) {
	rsp, err := client.ResponseError(cl.GetPerimeterPoliciesWithResponse(ctx, resourceId))
	if err != nil {
		return nil, fmt.Errorf("getting perimeter policies: %w", err)
	}
	if rsp.JSON200 == nil {
		return nil, fmt.Errorf("getting perimeter policies: unexpected empty response")
	}
	return *rsp.JSON200, nil
}

func OverwritePerimeterPolicy(ctx context.Context, cl *sam.ClientWithResponses, resourceId string, policyName string, memberEmails []string) error {
	var roles []string
	switch policyName {
	case "owner":
		roles = []string{"owner"}
	case "user":
		roles = []string{"user"}
	default:
		return fmt.Errorf("unknown policy name %q, expected \"owner\" or \"user\"", policyName)
	}
	req := sam.OverwritePerimeterPolicyJSONRequestBody{
		MemberEmails: memberEmails,
		Roles:        roles,
		Actions:      []string{},
	}
	if _, err := client.ResponseError(cl.OverwritePerimeterPolicyWithResponse(ctx, resourceId, policyName, req)); err != nil {
		return fmt.Errorf("overwriting perimeter policy %s: %w", policyName, err)
	}
	return nil
}

func DeletePerimeter(ctx context.Context, cl *sam.ClientWithResponses, resourceId string) error {
	rsp, err := client.ResponseError(cl.DeletePerimeterWithResponse(ctx, resourceId))
	if err != nil {
		if rsp != nil && (rsp.JSON404 != nil || rsp.StatusCode() == 404) {
			return nil
		}
		return fmt.Errorf("deleting perimeter: %w", err)
	}
	return nil
}

func SyncPerimeterGoogleGroup(ctx context.Context, cl *sam.ClientWithResponses, resourceId string) error {
	body := sam.SyncPerimeterGoogleGroupJSONRequestBody{}
	if _, err := client.ResponseError(cl.SyncPerimeterGoogleGroupWithResponse(ctx, resourceId, body)); err != nil {
		return fmt.Errorf("syncing perimeter google group: %w", err)
	}
	return nil
}

func GetPerimeterSyncStatus(ctx context.Context, cl *sam.ClientWithResponses, resourceId string) (*sam.SyncStatus, error) {
	rsp, err := client.ResponseError(cl.GetPerimeterSyncStatusWithResponse(ctx, resourceId))
	if err != nil {
		return nil, fmt.Errorf("getting perimeter sync status: %w", err)
	}
	return rsp.JSON200, nil
}

func NewSamClient(ctx context.Context, host string, useIdToken bool, impersonateServiceAccount string) (*sam.ClientWithResponses, error) {
	samUrl := fmt.Sprintf("%s/api/sam", host)
	httpClient, err := createHttpClient(ctx, samUrl, useIdToken, impersonateServiceAccount)
	if err != nil {
		return nil, err
	}
	return sam.NewClientWithResponses(samUrl, sam.WithHTTPClient(httpClient))
}
