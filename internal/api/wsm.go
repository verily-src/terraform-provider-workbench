// api contains calls to the workspace manager API.
package api

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/verily-src/terraform-provider-workbench/internal/client"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/wsm"
)

// GetWorkspace retrieves a workspace by its UUID.
func GetWorkspace(ctx context.Context, cl *wsm.ClientWithResponses, UUID string) (*wsm.WorkspaceDescription, error) {
	rsp, err := client.ResponseError(cl.GetWorkspaceWithResponse(ctx, UUID, nil))
	if err != nil {
		return nil, fmt.Errorf("getting workspace by uuid: %w", err)
	}
	return rsp.JSON200, nil
}

// CreateWorkspace starts a workspace creation flight. This is an asynchronous operation.
// The caller must poll for the job status using the job ID returned in the response.
func CreateWorkspace(ctx context.Context, cl *wsm.ClientWithResponses, request wsm.CreateWorkspaceV2JSONRequestBody) (string, error) {
	rsp, err := client.ResponseError(cl.CreateWorkspaceV2WithResponse(ctx, request))
	if err != nil {
		return "", fmt.Errorf("creating workspace: %w", err)
	}
	if rsp.JSON200 != nil {
		if rsp.JSON200.ErrorReport != nil {
			return "", fmt.Errorf("creating workspace, error: %s", rsp.JSON200.ErrorReport.Message)
		}
		return rsp.JSON200.JobReport.Id, nil
	}
	if rsp.JSON202 != nil {
		return rsp.JSON202.JobReport.Id, nil
	}
	return "", fmt.Errorf("creating workspace, unexpected response: %v", rsp.StatusCode())
}

// UpdateWorkspace updates a workspace with the given UUID.
func UpdateWorkspace(ctx context.Context, cl *wsm.ClientWithResponses, UUID string, request wsm.UpdateWorkspaceJSONRequestBody) (*wsm.WorkspaceDescription, error) {
	rsp, err := client.ResponseError(cl.UpdateWorkspaceWithResponse(ctx, UUID, request))
	if err != nil {
		return nil, fmt.Errorf("updating workspace: %v", err)
	}
	return rsp.JSON200, nil
}

// UpdateWorkspaceProperties updates the properties of a workspace with the given UUID.
func UpdateWorkspaceProperties(ctx context.Context, cl *wsm.ClientWithResponses, UUID string, request wsm.UpdateWorkspacePropertiesJSONRequestBody) error {
	if _, err := client.ResponseError(cl.UpdateWorkspacePropertiesWithResponse(ctx, UUID, request)); err != nil {
		return fmt.Errorf("updating workspace properties: %s", err)
	}
	return nil
}

// DeleteWorkspaceProperties deletes the properties of a workspace with the given UUID.
func DeleteWorkspaceProperties(ctx context.Context, cl *wsm.ClientWithResponses, UUID string, request wsm.DeleteWorkspacePropertiesJSONRequestBody) error {
	if _, err := client.ResponseError(cl.DeleteWorkspacePropertiesWithResponse(ctx, UUID, request)); err != nil {
		return fmt.Errorf("deleting workspace properties: %s", err)
	}
	return nil
}

// UpdateWorkspacePolicies updates the policies of a workspace with the given UUID.
func UpdateWorkspacePolicies(ctx context.Context, cl *wsm.ClientWithResponses, UUID string, request wsm.UpdatePoliciesJSONRequestBody) error {
	rsp, err := client.ResponseError(cl.UpdatePoliciesWithResponse(ctx, UUID, request))
	if err != nil {
		return fmt.Errorf("updating workspace policies: %s", err)
	}
	if rsp.JSON200.UpdateApplied {
		return nil
	}
	return fmt.Errorf("updating workspace policies, update not applied: %v", rsp.JSON200.Conflicts)
}

// DeleteWorkspace deletes a workspace with the given UUID. It starts a flight to delete the workspace and returns the job ID.
// The caller must poll for the job status using the job ID returned in the response.
func DeleteWorkspace(ctx context.Context, cl *wsm.ClientWithResponses, UUID string) (*string, error) {
	rsp, err := client.ResponseError(cl.DeleteWorkspaceV2WithResponse(ctx, UUID, wsm.DeleteWorkspaceV2JSONRequestBody{
		JobControl: wsm.JobControl{
			Id: uuid.New().String(),
		},
	}))
	if rsp.JSON200 != nil {
		return client.Ptr(rsp.JSON200.JobReport.Id), nil
	}
	if rsp.JSON202 != nil {
		return client.Ptr(rsp.JSON202.JobReport.Id), nil
	}
	if rsp.JSON404 != nil {
		// Workspace not found
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("Unable to delete workspace: %s", err)
	}
	return nil, fmt.Errorf("deleting workspace, unexpected response: %v", rsp.StatusCode())
}

func SetRole(ctx context.Context, cl *wsm.ClientWithResponses, UUID string, request wsm.SetRoleJSONRequestBody) error {
	if _, err := client.ResponseError(cl.SetRoleWithResponse(ctx, UUID, request)); err != nil {
		return fmt.Errorf("setting role for workspace %s: %v", UUID, err)
	}
	return nil
}

func GetRoles(ctx context.Context, cl *wsm.ClientWithResponses, UUID string) (*wsm.RoleBindingList, error) {
	rsp, err := client.ResponseError(cl.GetRolesWithResponse(ctx, UUID))
	if err != nil {
		return nil, fmt.Errorf("getting roles for workspace %s: %v", UUID, err)
	}
	return rsp.JSON200, nil
}

// CreateFolder creates a folder in the workspace.
func CreateFolder(ctx context.Context, cl *wsm.ClientWithResponses, UUID string, request wsm.CreateFolderJSONRequestBody) (*wsm.Folder, error) {
	rsp, err := client.ResponseError(cl.CreateFolderWithResponse(ctx, UUID, request))
	if err != nil || rsp.JSON200 == nil {
		return nil, fmt.Errorf("creating folder in workspace %s: %v", UUID, err)
	}
	return rsp.JSON200, nil
}

// GetFolder retrieves a folder by its ID in the specified workspace.
func GetFolder(ctx context.Context, cl *wsm.ClientWithResponses, workspaceId, folderId string) (*wsm.Folder, error) {
	fId, err := uuid.Parse(folderId)
	if err != nil {
		return nil, fmt.Errorf("invalid folder ID %s: %v", folderId, err)
	}
	// DO NOT pass nil as reqEditors - will panic
	rsp, err := client.ResponseError(cl.GetFolderWithResponse(ctx, workspaceId, fId))
	if err != nil {
		return nil, fmt.Errorf("getting folder %s in workspace %s: %v", folderId, workspaceId, err)
	}
	return rsp.JSON200, nil
}

// UpdateFolder updates metadata of a folder in the specified workspace.
func UpdateFolder(ctx context.Context, cl *wsm.ClientWithResponses, workspaceId, folderId string, request wsm.UpdateFolderJSONRequestBody) (*wsm.Folder, error) {
	fId, err := uuid.Parse(folderId)
	if err != nil {
		return nil, fmt.Errorf("invalid folder ID %s: %v", folderId, err)
	}
	rsp, err := client.ResponseError(cl.UpdateFolderWithResponse(ctx, workspaceId, fId, request))
	if err != nil {
		return nil, fmt.Errorf("updating folder %s in workspace %s: %v", folderId, workspaceId, err)
	}
	if rsp.JSON200 == nil {
		return nil, fmt.Errorf("folder %s not found in workspace %s", folderId, workspaceId)
	}
	return rsp.JSON200, nil
}

// UpdateFolderProperties updates the properties of a folder in the specified workspace. Only properties with keys in request are updated. Properties with keys not in request are not updated.
func UpdateFolderProperties(ctx context.Context, cl *wsm.ClientWithResponses, workspaceId, folderId string, request wsm.UpdateFolderPropertiesJSONRequestBody) error {
	fId, err := uuid.Parse(folderId)
	if err != nil {
		return fmt.Errorf("invalid folder ID %s: %v", folderId, err)
	}
	if _, err := client.ResponseError(cl.UpdateFolderPropertiesWithResponse(ctx, workspaceId, fId, request)); err != nil {
		return fmt.Errorf("updating folder properties %s in workspace %s: %v", folderId, workspaceId, err)
	}
	return nil
}

// DeleteFolderProperties deletes the properties of a folder in the specified workspace. Only properties with keys in request are deleted. Properties with keys not in request are not deleted.
func DeleteFolderProperties(ctx context.Context, cl *wsm.ClientWithResponses, workspaceId, folderId string, request wsm.DeleteFolderPropertiesJSONRequestBody) error {
	fId, err := uuid.Parse(folderId)
	if err != nil {
		return fmt.Errorf("invalid folder ID %s: %v", folderId, err)
	}
	if _, err := client.ResponseError(cl.DeleteFolderPropertiesWithResponse(ctx, workspaceId, fId, request)); err != nil {
		return fmt.Errorf("deleting folder properties %s in workspace %s: %v", folderId, workspaceId, err)
	}
	return nil
}

// DeleteFolderAsync deletes a folder and resources in it in the specified workspace asynchronously.
func DeleteFolderAsync(ctx context.Context, cl *wsm.ClientWithResponses, workspaceId, folderId string) (*string, error) {
	fId, err := uuid.Parse(folderId)
	if err != nil {
		return nil, fmt.Errorf("invalid folder ID %s: %v", folderId, err)
	}
	rsp, err := client.ResponseError(cl.DeleteFolderAsyncWithResponse(ctx, workspaceId, fId))
	if err != nil {
		return nil, fmt.Errorf("deleting folder: %w", err)
	}
	if rsp.JSON200 != nil {
		return client.Ptr(rsp.JSON200.JobReport.Id), nil
	}
	if rsp.JSON202 != nil {
		return client.Ptr(rsp.JSON202.JobReport.Id), nil
	}
	return nil, fmt.Errorf("deleting folder, unexpected response: %v", rsp.StatusCode())
}

// GetControlledGcsBucket retrieves a controlled GCS bucket by its resource ID in the specified workspace.
func GetControlledGcsBucket(ctx context.Context, cl *wsm.ClientWithResponses, workspaceId, resourceId string) (*wsm.GetControlledGcpGcsBucketResponse, error) {
	rId, err := uuid.Parse(resourceId)
	if err != nil {
		return nil, fmt.Errorf("invalid resource ID %s: %v", resourceId, err)
	}
	rsp, err := client.ResponseError(cl.GetBucketWithResponse(ctx, workspaceId, rId))
	if err != nil {
		return nil, fmt.Errorf("getting controlled GCS bucket: %w", err)
	}
	return rsp.JSON200, nil
}

// CreateControlledGcsBucket creates a controlled GCS bucket in the specified workspace.
func CreateControlledGcsBucket(ctx context.Context, cl *wsm.ClientWithResponses, workspaceId string, request wsm.CreateControlledGcpGcsBucketRequestBody) (*wsm.CreatedControlledGcpGcsBucketResponse, error) {
	rsp, err := client.ResponseError(cl.CreateBucketWithResponse(ctx, workspaceId, request))
	if err != nil {
		return nil, fmt.Errorf("creating controlled GCS bucket: %w", err)
	}
	return rsp.JSON200, nil
}

// UpdateControlledGcsBucket updates a controlled GCS bucket in the specified workspace.
func UpdateControlledGcsBucket(ctx context.Context, cl *wsm.ClientWithResponses, workspaceId, resourceId string, request wsm.UpdateGcsBucketJSONRequestBody) (*wsm.UpdateControlledGcpGcsBucketResponse, error) {
	rId, err := uuid.Parse(resourceId)
	if err != nil {
		return nil, fmt.Errorf("parsing resource ID %s: %v", resourceId, err)
	}
	rsp, err := client.ResponseError(cl.UpdateGcsBucketWithResponse(ctx, workspaceId, rId, request))
	if err != nil {
		return nil, fmt.Errorf("updating controlled GCS bucket: %w", err)
	}
	return rsp.JSON200, nil
}

// DeleteControlledGcsBucketAsync deletes a controlled GCS bucket asynchronously in the specified workspace.
// It returns the job ID for tracking the deletion status.
func DeleteControlledGcsBucketAsync(ctx context.Context, cl *wsm.ClientWithResponses, workspaceId, resourceId string) (*string, error) {
	rId, err := uuid.Parse(resourceId)
	if err != nil {
		return nil, fmt.Errorf("invalid resource ID %s: %v", resourceId, err)
	}
	rsp, err := client.ResponseError(cl.DeleteBucketWithResponse(ctx, workspaceId, rId, wsm.DeleteControlledGcpGcsBucketRequest{
		JobControl: wsm.JobControl{
			Id: uuid.New().String(),
		},
	}))
	if err != nil {
		return nil, fmt.Errorf("deleting controlled GCS bucket: %w", err)
	}
	if rsp.JSON200 != nil {
		return client.Ptr(rsp.JSON200.JobReport.Id), nil
	}
	if rsp.JSON202 != nil {
		return client.Ptr(rsp.JSON202.JobReport.Id), nil
	}
	return nil, fmt.Errorf("deleting gcs bucket, unexpected response: %v", rsp.StatusCode())
}

// NewWSMClient creates a new WSM client with the given host and context.
func NewWSMClient(ctx context.Context, host string) (*wsm.ClientWithResponses, error) {
	wsmUrl := fmt.Sprintf("%s/api/wsm", host)

	httpClient, err := createHttpClient(ctx, wsmUrl)
	if err != nil {
		return nil, err
	}
	return wsm.NewClientWithResponses(wsmUrl, wsm.WithHTTPClient(httpClient))
}
