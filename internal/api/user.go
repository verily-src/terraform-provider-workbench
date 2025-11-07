// api contains calls to the workspace manager API.
package api

import (
	"context"
	"fmt"

	"github.com/verily-src/terraform-provider-workbench/internal/client"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/user"
)

// CreateGroup creates a workbench managed group.
func CreateGroup(ctx context.Context, cl *user.ClientWithResponses, orgUFID string, request user.CreateGroupJSONRequestBody) error {
	if _, err := client.ResponseError(cl.CreateGroupWithResponse(ctx, *uxid(orgUFID), request)); err != nil {
		return fmt.Errorf("creating group: %w", err)
	}
	return nil
}

// DescribeGroup retrieves a workbench managed group by its name.
func DescribeGroup(ctx context.Context, cl *user.ClientWithResponses, groupName string, orgUFID string) (*user.GroupDescriptionAndRoles, error) {
	rsp, err := client.ResponseError(cl.DescribeGroupWithResponse(ctx, groupName, &user.DescribeGroupParams{
		OrgIdQueryParam: uxid(orgUFID),
	}))
	if err != nil {
		return nil, fmt.Errorf("getting group: %w", err)
	}
	return rsp.JSON200, nil
}

// DeleteGroup deletes a workbench managed group.
func DeleteGroup(ctx context.Context, cl *user.ClientWithResponses, groupName string, orgUFID string) error {
	rsp, err := client.ResponseError(cl.DeleteGroupWithResponse(ctx, *uxid(orgUFID), groupName))
	if rsp.JSON404 != nil {
		// Group not found, return nil
		return nil
	}
	if err != nil {
		return fmt.Errorf("deleting group: %w", err)
	}
	return nil
}

// UpdateGroup updates a workbench managed group.
func UpdateGroup(ctx context.Context, cl *user.ClientWithResponses, groupName string, orgUFID string, request user.UpdateGroupRequest) error {
	if _, err := client.ResponseError(cl.UpdateGroupWithResponse(ctx, *uxid(orgUFID), groupName, request)); err != nil {
		return fmt.Errorf("updating group: %w", err)
	}
	return nil
}

// SyncGroup synchronizes a workbench managed group with the google group.
func SyncGroup(ctx context.Context, cl *user.ClientWithResponses, groupName string, orgUFID string) error {
	if _, err := client.ResponseError(cl.SyncGroupWithResponse(ctx, *uxid(orgUFID), groupName)); err != nil {
		return fmt.Errorf("syncing group: %w", err)
	}
	return nil
}

// SetGroupRole grants a role to a principal in a workbench managed group.
func SetGroupRole(ctx context.Context, cl *user.ClientWithResponses, groupName user.GroupNameParam, orgID *user.OrgIdQueryParam, request user.SetAccessRequest) error {
	var params *user.SetGroupAccessParams
	if orgID != nil {
		params = &user.SetGroupAccessParams{
			OrgIdQueryParam: orgID,
		}
	}
	if _, err := client.ResponseError(cl.SetGroupAccessWithResponse(ctx, groupName, params, request)); err != nil {
		return fmt.Errorf("setting group role for %s: %w", groupName, err)
	}
	return nil
}

// GetGroupRoles retrieves the roles of a workbench managed group.
func GetGroupRoles(ctx context.Context, cl *user.ClientWithResponses, groupName user.GroupNameParam, orgID *user.OrgIdQueryParam) (*user.GroupMemberList, error) {
	var params *user.ListGroupMembershipParams
	if orgID != nil {
		params = &user.ListGroupMembershipParams{
			OrgIdQueryParam: orgID,
		}
	}
	rsp, err := client.ResponseError(cl.ListGroupMembershipWithResponse(ctx, groupName, params))
	if err != nil {
		return nil, fmt.Errorf("getting group roles: %w", err)
	}
	if rsp.JSON404 != nil {
		return nil, fmt.Errorf("group %s not found", groupName)
	}
	return rsp.JSON200, nil
}

func uxid(s string) *string {
	if s == "" {
		return nil
	}
	uxid := fmt.Sprintf("~%s", s)
	return &uxid
}

// NewUserClient creates a new user manager client with the given host and context.
func NewUserClient(ctx context.Context, host string, useIdToken bool, impersonateServiceAccount string) (*user.ClientWithResponses, error) {
	userUrl := fmt.Sprintf("%s/api/user", host)
	httpClient, err := createHttpClient(ctx, userUrl, useIdToken, impersonateServiceAccount)
	if err != nil {
		return nil, err
	}
	return user.NewClientWithResponses(userUrl, user.WithHTTPClient(httpClient))
}
