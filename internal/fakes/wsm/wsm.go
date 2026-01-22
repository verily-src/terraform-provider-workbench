// Package wsm provides a workspace-manager fake at the service API layer.
package wsm

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/verily-src/terraform-provider-workbench/internal/client"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/wsm"
)

// New creates a new workspace-manager Service struct.
func New() *Service {
	return &Service{
		workspaces:   make([]*workspaceState, 0),
		createJobIds: make([]string, 0),
		deleteJobMap: make(map[string]string),
	}
}

type workspaceState struct {
	wsm.WorkspaceDescription
	roles   map[wsm.IamRole][]string   // role -> members
	folders []wsm.Folder               // Folders in the workspace
	buckets []wsm.GcpGcsBucketResource // Buckets in the workspace
}

func (ws *workspaceState) setRole(op wsm.SetAccessOperation, role wsm.IamRole, member string) error {
	switch op {
	case wsm.GRANT:
		ws.grantRole(role, member)
	case wsm.REVOKE:
		ws.revokeRole(role, member)
	default:
		return fmt.Errorf("unsupported operation %s", op)
	}
	return nil
}

func (ws *workspaceState) grantRole(role wsm.IamRole, member string) {
	if ws.roles == nil {
		ws.roles = make(map[wsm.IamRole][]string)
	}
	ws.roles[role] = append(ws.roles[role], member)
}

func (ws *workspaceState) revokeRole(role wsm.IamRole, member string) {
	if ws.roles == nil {
		return
	}
	if members, ok := ws.roles[role]; ok {
		for i, m := range members {
			if m != member {
				continue
			}
			ws.roles[role] = slices.Delete(members, i, i+1)
			if len(ws.roles[role]) == 0 {
				delete(ws.roles, role) // Remove the role if no members left
			}
			return
		}
	}
}

// Service implements the APIs for a fake workspace-manager service.
type Service struct {
	sync.Mutex
	workspaces   []*workspaceState
	createJobIds []string
	deleteJobMap map[string]string
}

func (s *Service) RegisterHTTP(e *echo.Echo) {
	g := e.Group("/api/wsm")

	// getWorkspaceByUserFacingId
	g.GET("/api/workspaces/v1/workspaceByUserFacingId/:userFacingId", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()

		userFacingId := c.Param("userFacingId")
		ws, ok := s.findByUfid(userFacingId)
		if !ok {
			return jsonError(c, http.StatusNotFound, "workspace %s not found", userFacingId)
		}
		return c.JSON(http.StatusOK, ws)
	})

	// getWorkspace
	g.GET("/api/workspaces/v1/:workspaceId", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()

		workspaceId := c.Param("workspaceId")
		ws, ok := s.find(workspaceId)
		if !ok {
			return jsonError(c, http.StatusNotFound, "workspace %s not found", workspaceId)
		}
		return c.JSON(http.StatusOK, ws)
	})

	// listWorkspaces
	g.GET("/api/workspaces/v1", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()

		ws := make([]wsm.WorkspaceDescription, 0, len(s.workspaces))
		for _, w := range s.workspaces {
			ws = append(ws, w.WorkspaceDescription)
		}

		return c.JSON(http.StatusOK, &wsm.WorkspaceDescriptionList{
			Workspaces: ws,
		})
	})

	// createWorkspaceV2
	g.POST("/api/workspaces/v2", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()

		var req wsm.CreateWorkspaceV2Request
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			return c.NoContent(http.StatusBadRequest)
		}

		id := req.Id.String()
		if _, ok := s.find(id); ok {
			return jsonError(c, http.StatusConflict, "workspace %s already exists", id)
		}

		orgId, err := uuid.Parse(*req.OrganizationId)
		if err != nil {
			return jsonError(c, http.StatusBadRequest, "invalid org id %s: %v", *req.OrganizationId, err)
		}
		crgId, err := uuid.Parse(*req.CloudResourceGroupId)
		if err != nil {
			return jsonError(c, http.StatusBadRequest, "invalid pod id %s: %v", *req.CloudResourceGroupId, err)
		}
		userFacingId := "a" + req.Id.String()
		if req.UserFacingId != nil && *req.UserFacingId != "" {
			userFacingId = *req.UserFacingId
		}

		var policies *[]wsm.WsmPolicyInput
		if req.Policies != nil {
			policies = &req.Policies.Inputs
		}

		now := time.Now()
		s.workspaces = append(s.workspaces, &workspaceState{
			WorkspaceDescription: wsm.WorkspaceDescription{
				Id:              req.Id,
				UserFacingId:    userFacingId,
				DisplayName:     req.DisplayName,
				Description:     req.Description,
				HighestRole:     wsm.IamRoleOWNER,
				Stage:           req.Stage,
				OrgId:           &orgId,
				CrgId:           &crgId,
				Properties:      req.Properties,
				CreatedDate:     now,
				CreatedBy:       "testuser",
				LastUpdatedDate: now,
				LastUpdatedBy:   "testuser",
				OperationState:  &wsm.OperationState{State: wsm.CREATING},
				Policies:        policies,
				GcpContext: &wsm.GcpContext{
					ProjectId: "test-gcp-project-" + req.Id.String()[:8],
				},
			},
		})
		jobID := uuid.New().String()
		s.createJobIds = append(s.createJobIds, jobID)
		return c.JSON(http.StatusAccepted, &wsm.CreateWorkspaceV2Response{
			JobReport: &wsm.JobReport{
				Id:         jobID,
				StatusCode: 202,
				Status:     wsm.JobReportStatusRUNNING,
			},
		})
	})

	// getCreateWorkspaceResult
	g.GET("/api/workspaces/v2/result/:jobId", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()

		jobID := c.Param("jobId")
		job := s.findCreateJob(jobID)
		if job == nil {
			return jsonError(c, http.StatusNotFound, "job %s not found", jobID)
		}
		s.createJobIds = remove(s.createJobIds, jobID)
		return c.JSON(http.StatusOK, &wsm.CreateWorkspaceV2Result{
			JobReport: job,
		})
	})

	// updateWorkspace
	g.PATCH("/api/workspaces/v1/:workspaceId", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()

		var req wsm.UpdateWorkspaceRequestBody
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			return c.NoContent(http.StatusBadRequest)
		}

		workspaceId := c.Param("workspaceId")
		ws, ok := s.find(workspaceId)
		if !ok {
			return jsonError(c, http.StatusNotFound, "workspace %s not found", workspaceId)
		}

		if req.UserFacingId != nil {
			ws.UserFacingId = *req.UserFacingId
		}
		if req.DisplayName != nil {
			ws.DisplayName = req.DisplayName
		}
		if req.Description != nil {
			ws.Description = req.Description
		}

		if ok := s.findAndReplace(workspaceId, ws.WorkspaceDescription); !ok {
			return jsonError(c, http.StatusNotFound, "workspace %s not found", workspaceId)
		}

		return c.JSON(http.StatusOK, ws)
	})

	// update workspace properties
	g.POST("/api/workspaces/v1/:workspaceId/properties", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()
		var req wsm.Properties
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			return c.NoContent(http.StatusBadRequest)
		}
		workspaceId := c.Param("workspaceId")
		ws, ok := s.find(workspaceId)
		if !ok {
			return jsonError(c, http.StatusNotFound, "workspace %s not found", workspaceId)
		}
		newProperties := mergeProperties(*ws.Properties, req)
		ws.Properties = &newProperties
		if ok := s.findAndReplace(workspaceId, ws.WorkspaceDescription); !ok {
			return jsonError(c, http.StatusNotFound, "workspace %s not found", workspaceId)
		}
		return c.JSON(http.StatusNoContent, nil)
	})

	// remove workspace properties
	g.PATCH("/api/workspaces/v1/:workspaceId/properties", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()
		var req wsm.PropertyKeys
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			return c.NoContent(http.StatusBadRequest)
		}
		workspaceId := c.Param("workspaceId")
		ws, ok := s.find(workspaceId)
		if !ok {
			return jsonError(c, http.StatusNotFound, "workspace %s not found", workspaceId)
		}
		newProperties := removeProperties(*ws.Properties, req)
		ws.Properties = &newProperties

		if ok := s.findAndReplace(workspaceId, ws.WorkspaceDescription); !ok {
			return jsonError(c, http.StatusNotFound, "workspace %s not found", workspaceId)
		}
		return c.JSON(http.StatusNoContent, nil)
	})

	// Async deleteWorkspace
	g.DELETE("/api/workspaces/v2/:workspaceId", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()

		workspaceID := c.Param("workspaceId")
		if _, ok := s.find(workspaceID); !ok {
			return jsonError(c, http.StatusNotFound, "workspace %s not found", workspaceID)
		}

		jobID := uuid.New().String()
		s.deleteJobMap[jobID] = workspaceID
		return c.JSON(http.StatusAccepted, &wsm.JobResult{
			JobReport: wsm.JobReport{
				Id:         jobID,
				StatusCode: 202,
				Status:     wsm.JobReportStatusRUNNING,
			},
		})
	})

	// getDeleteWorkspaceResult
	g.GET("/api/workspaces/v2/:workspaceId/delete-result/:jobId", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()

		jobID := c.Param("jobId")
		wID := c.Param("workspaceId")
		if workspaceID, ok := s.deleteJobMap[jobID]; !ok || workspaceID != wID {
			return jsonError(c, http.StatusNotFound, "delete workspace %s job %s not found", wID, jobID)
		}
		if ok := s.removeWorkspace(wID); !ok {
			return jsonError(c, http.StatusNotFound, "workspace %s not found", wID)
		}
		delete(s.deleteJobMap, jobID)

		return c.JSON(http.StatusOK, &wsm.JobResult{
			JobReport: wsm.JobReport{
				Id:         jobID,
				StatusCode: 200,
				Status:     wsm.JobReportStatusSUCCEEDED,
			},
		})
	})

	// setRole
	g.POST("/api/workspaces/v1/:workspaceId/roles", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()
		var req wsm.SetAccessRequest
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			return c.NoContent(http.StatusBadRequest)
		}

		workspaceId := c.Param("workspaceId")
		ws, ok := s.find(workspaceId)
		if !ok {
			return jsonError(c, http.StatusNotFound, "workspace %s not found", workspaceId)
		}

		var err error
		switch p := req.Principal; true {
		case p.UserPrincipal != nil:
			err = ws.setRole(req.Operation, req.Role, req.Principal.UserPrincipal.Email)
		default:
			return jsonError(c, http.StatusNotImplemented, "unsupported principal type")
		}
		if err != nil {
			return jsonError(c, http.StatusInternalServerError, "error setting role: %v", err)
		}

		return c.JSON(http.StatusNoContent, nil)
	})

	// getRoles
	g.GET("/api/workspaces/v1/:workspaceId/roles", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()

		workspaceId := c.Param("workspaceId")
		ws, ok := s.find(workspaceId)
		if !ok {
			return jsonError(c, http.StatusNotFound, "workspace %s not found", workspaceId)
		}

		roleBindings := make([]wsm.RoleBinding, 0, len(ws.roles))
		for role, members := range ws.roles {
			roleBindings = append(roleBindings, wsm.RoleBinding{
				Role:    role,
				Members: &members,
			})
		}

		return c.JSON(http.StatusOK, wsm.RoleBindingList(roleBindings))
	})

	// createFolder
	g.POST("/api/workspaces/v1/:workspaceId/folders", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()

		var req wsm.CreateFolderJSONRequestBody
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			return c.NoContent(http.StatusBadRequest)
		}

		workspaceId := c.Param("workspaceId")
		ws, ok := s.find(workspaceId)
		if !ok {
			return jsonError(c, http.StatusNotFound, "workspace %s not found", workspaceId)
		}

		folder := wsm.Folder{
			Id:              uuid.New(),
			DisplayName:     req.DisplayName,
			Description:     req.Description,
			Properties:      req.Properties,
			LastUpdatedDate: time.Now(),
			LastUpdatedBy:   "testuser",
			CreatedDate:     time.Now(),
			CreatedBy:       "testuser",
		}
		ws.folders = append(ws.folders, folder)

		return c.JSON(http.StatusOK, folder)
	})

	// get folder
	g.GET("/api/workspaces/v1/:workspaceId/folders/:folderId", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()

		workspaceId := c.Param("workspaceId")
		folderId := c.Param("folderId")

		ws, ok := s.find(workspaceId)
		if !ok {
			return jsonError(c, http.StatusNotFound, "workspace %s not found", workspaceId)
		}

		for _, folder := range ws.folders {
			if folder.Id.String() == folderId {
				return c.JSON(http.StatusOK, folder)
			}
		}

		return jsonError(c, http.StatusNotFound, "folder %s not found in workspace %s", folderId, workspaceId)
	})

	// update folder
	g.PATCH("/api/workspaces/v1/:workspaceId/folders/:folderId", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()

		var req wsm.UpdateFolderJSONRequestBody
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			return c.NoContent(http.StatusBadRequest)
		}

		workspaceId := c.Param("workspaceId")
		folderId := c.Param("folderId")

		ws, ok := s.find(workspaceId)
		if !ok {
			return jsonError(c, http.StatusNotFound, "workspace %s not found", workspaceId)
		}

		for i, folder := range ws.folders {
			if folder.Id.String() == folderId {
				if req.DisplayName != nil {
					folder.DisplayName = *req.DisplayName
				}
				if req.Description != nil {
					folder.Description = req.Description
				}
				ws.folders[i] = folder
				return c.JSON(http.StatusOK, folder)
			}
		}

		return jsonError(c, http.StatusNotFound, "folder %s not found in workspace %s", folderId, workspaceId)
	})

	// delete folder async
	g.POST("/api/workspaces/v1/:workspaceId/folders/:folderId", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()

		workspaceId := c.Param("workspaceId")
		folderId := c.Param("folderId")

		ws, ok := s.find(workspaceId)
		if !ok {
			return jsonError(c, http.StatusNotFound, "workspace %s not found", workspaceId)
		}

		for _, folder := range ws.folders {
			if folder.Id.String() == folderId {
				jobID := uuid.New().String()
				s.deleteJobMap[jobID] = folderId
				return c.JSON(http.StatusAccepted, &wsm.JobResult{
					JobReport: wsm.JobReport{
						Id:         jobID,
						StatusCode: 202,
						Status:     wsm.JobReportStatusRUNNING,
					},
				})
			}
		}
		return jsonError(c, http.StatusNotFound, "folder %s not found in workspace %s", folderId, workspaceId)

	})

	g.GET("/api/workspaces/v1/:workspaceId/folders/:folderId/result/:jobId", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()

		jobID := c.Param("jobId")
		workspaceID := c.Param("workspaceId")
		folderID := c.Param("folderId")
		if fID, ok := s.deleteJobMap[jobID]; !ok || fID != folderID {
			return jsonError(c, http.StatusNotFound, "delete folder %s job %s not found", folderID, jobID)
		}
		s.removeFolder(workspaceID, folderID)
		delete(s.deleteJobMap, jobID)

		return c.JSON(http.StatusOK, &wsm.JobResult{
			JobReport: wsm.JobReport{
				Id:         jobID,
				StatusCode: 200,
				Status:     wsm.JobReportStatusSUCCEEDED,
			},
		})

	})

	// set folder properties
	g.POST("/api/workspaces/v1/:workspaceId/folders/:folderId/properties", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()

		var req wsm.Properties
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			return c.NoContent(http.StatusBadRequest)
		}

		workspaceId := c.Param("workspaceId")
		folderId := c.Param("folderId")

		ws, ok := s.find(workspaceId)
		if !ok {
			return jsonError(c, http.StatusNotFound, "workspace %s not found", workspaceId)
		}

		for i, folder := range ws.folders {
			if folder.Id.String() == folderId {
				newProperties := mergeProperties(*folder.Properties, req)
				folder.Properties = &newProperties
				ws.folders[i] = folder
				return c.JSON(http.StatusNoContent, nil)
			}
		}

		return jsonError(c, http.StatusNotFound, "folder %s not found in workspace %s", folderId, workspaceId)
	})

	// remove folder properties
	g.PATCH("/api/workspaces/v1/:workspaceId/folders/:folderId/properties", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()

		var req wsm.PropertyKeys
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			return c.NoContent(http.StatusBadRequest)
		}

		workspaceId := c.Param("workspaceId")
		folderId := c.Param("folderId")

		ws, ok := s.find(workspaceId)
		if !ok {
			return jsonError(c, http.StatusNotFound, "workspace %s not found", workspaceId)
		}

		for i, folder := range ws.folders {
			if folder.Id.String() == folderId {
				newProperties := removeProperties(*folder.Properties, req)
				folder.Properties = &newProperties
				ws.folders[i] = folder
				return c.JSON(http.StatusNoContent, nil)
			}
		}

		return jsonError(c, http.StatusNotFound, "folder %s not found in workspace %s", folderId, workspaceId)
	})

	// create controlled GCS bucket
	g.POST("/api/workspaces/v1/:workspaceId/resources/controlled/gcp/buckets", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()

		var req wsm.CreateControlledGcpGcsBucketRequestBody
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			return c.NoContent(http.StatusBadRequest)
		}

		workspaceId := c.Param("workspaceId")
		ws, ok := s.find(workspaceId)
		if !ok {
			return jsonError(c, http.StatusNotFound, "workspace %s not found", workspaceId)
		}

		wID, err := uuid.Parse(workspaceId)
		if err != nil {
			return jsonError(c, http.StatusBadRequest, "invalid workspace id %s: %v", workspaceId, err)
		}

		rID := uuid.New()
		location := req.GcsBucket.Location
		if req.GcsBucket.Location == nil {
			// This is retrieved from the workspace location, but as a fake, we are just
			// defaulting to us-central1.
			location = client.Ptr("us-central1")
		}
		bucket := wsm.GcpGcsBucketResource{
			Metadata: wsm.ResourceMetadata{
				WorkspaceId:         wID,
				ResourceId:          rID,
				Name:                *req.Common.Name,
				DisplayName:         req.Common.DisplayName,
				Description:         req.Common.Description,
				FolderId:            req.Common.FolderId,
				ResourceType:        wsm.GCSBUCKET,
				StewardshipType:     wsm.CONTROLLED,
				CloningInstructions: client.Ptr(req.Common.CloningInstructions),
				ControlledResourceMetadata: &wsm.ControlledResourceMetadata{
					AccessScope:          client.Ptr(req.Common.AccessScope),
					ManagedBy:            client.Ptr(req.Common.ManagedBy),
					PrivateResourceState: client.Ptr(wsm.PrivateResourceStateNOTAPPLICABLE),
					Region:               location,
				},
				LastUpdatedDate: time.Now(),
				LastUpdatedBy:   "testuser",
				CreatedDate:     time.Now(),
				CreatedBy:       "testuser",
				State:           client.Ptr(wsm.READY),
				CloudPlatform:   client.Ptr(wsm.GCP),
			},
			Attributes: wsm.GcpGcsBucketAttributes{
				BucketName: *req.GcsBucket.Name,
			},
		}
		ws.buckets = append(ws.buckets, bucket)

		rsp := &wsm.CreatedControlledGcpGcsBucket{
			GcpBucket:  bucket,
			ResourceId: rID,
		}

		return c.JSON(http.StatusOK, rsp)
	})

	// get controlled GCS bucket
	g.GET("/api/workspaces/v1/:workspaceId/resources/controlled/gcp/buckets/:resourceId", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()

		workspaceId := c.Param("workspaceId")
		resourceId := c.Param("resourceId")

		ws, ok := s.find(workspaceId)
		if !ok {
			return jsonError(c, http.StatusNotFound, "workspace %s not found", workspaceId)
		}

		for _, bucket := range ws.buckets {
			if bucket.Metadata.ResourceId.String() == resourceId {
				return c.JSON(http.StatusOK, bucket)
			}
		}

		return jsonError(c, http.StatusNotFound, "bucket %s not found in workspace %s", resourceId, workspaceId)
	})

	// update a controlled GCS bucket
	g.PATCH("/api/workspaces/v1/:workspaceId/resources/controlled/gcp/buckets/:resourceId", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()

		var req wsm.UpdateControlledGcpGcsBucketRequestBody
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			return c.NoContent(http.StatusBadRequest)
		}

		workspaceId := c.Param("workspaceId")
		resourceId := c.Param("resourceId")

		ws, ok := s.find(workspaceId)
		if !ok {
			return jsonError(c, http.StatusNotFound, "workspace %s not found", workspaceId)
		}

		for i, bucket := range ws.buckets {
			if bucket.Metadata.ResourceId.String() == resourceId {
				if req.Name != nil {
					bucket.Metadata.Name = *req.Name
				}
				if req.DisplayName != nil {
					bucket.Metadata.DisplayName = req.DisplayName
				}
				if req.Description != nil {
					bucket.Metadata.Description = req.Description
				}
				if req.UpdateFolderId != nil {
					bucket.Metadata.FolderId = req.UpdateFolderId.FolderId
				}
				if req.UpdateParameters != nil && req.UpdateParameters.CloningInstructions != nil {
					bucket.Metadata.CloningInstructions = req.UpdateParameters.CloningInstructions
				}
				ws.buckets[i] = bucket
				return c.JSON(http.StatusOK, bucket)
			}
		}

		return jsonError(c, http.StatusNotFound, "bucket %s not found in workspace %s", resourceId, workspaceId)
	})

	// delete controlled GCS bucket async
	g.POST("/api/workspaces/v1/:workspaceId/resources/controlled/gcp/buckets/:resourceId", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()

		workspaceId := c.Param("workspaceId")
		resourceId := c.Param("resourceId")

		ws, ok := s.find(workspaceId)
		if !ok {
			return jsonError(c, http.StatusNotFound, "workspace %s not found", workspaceId)
		}

		for _, bucket := range ws.buckets {
			if bucket.Metadata.ResourceId.String() == resourceId {
				jobID := uuid.New().String()
				s.deleteJobMap[jobID] = workspaceId + ":" + resourceId
				return c.JSON(http.StatusAccepted, &wsm.JobResult{
					JobReport: wsm.JobReport{
						Id:         jobID,
						StatusCode: 202,
						Status:     wsm.JobReportStatusRUNNING,
					},
				})
			}
		}

		return jsonError(c, http.StatusNotFound, "bucket %s not found in workspace %s", resourceId, workspaceId)
	})
	// get delete controlled GCS bucket result
	g.GET("/api/workspaces/v1/:workspaceId/resources/controlled/gcp/buckets/delete-result/:jobId", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()

		jobId := c.Param("jobId")
		parts := strings.Split(s.deleteJobMap[jobId], ":")
		if len(parts) != 2 {
			return jsonError(c, http.StatusNotFound, "delete job %s not found", jobId)
		}
		workspaceId := parts[0]
		resourceId := parts[1]

		ws, ok := s.find(workspaceId)
		if !ok {
			return jsonError(c, http.StatusNotFound, "workspace %s not found", workspaceId)
		}
		for i, bucket := range ws.buckets {
			if bucket.Metadata.ResourceId.String() == resourceId {
				ws.buckets = append(ws.buckets[:i], ws.buckets[i+1:]...)
				delete(s.deleteJobMap, jobId)
				return c.JSON(http.StatusOK, &wsm.JobResult{
					JobReport: wsm.JobReport{
						Id:         jobId,
						StatusCode: 200,
						Status:     wsm.JobReportStatusSUCCEEDED,
					},
				})
			}
		}

		return jsonError(c, http.StatusNotFound, "bucket %s not found in workspace %s", resourceId, workspaceId)
	})
}

func jsonError(c echo.Context, code int, format string, a ...interface{}) error {
	return c.JSON(code, client.NewApiError(code, fmt.Sprintf(format, a...)))
}

func (s *Service) find(workspaceID string) (*workspaceState, bool) {
	for _, ws := range s.workspaces {
		if ws.Id.String() == workspaceID {
			return ws, true
		}
	}
	return nil, false
}

func (s *Service) findByUfid(ufid string) (*workspaceState, bool) {
	for _, ws := range s.workspaces {
		if ws.UserFacingId == ufid {
			return ws, true
		}
	}
	return nil, false
}

func (s *Service) findAndReplace(workspaceID string, ws wsm.WorkspaceDescription) bool {
	for i, w := range s.workspaces {
		if w.Id.String() == workspaceID {
			s.workspaces[i].WorkspaceDescription = ws
			return true
		}
	}
	return false
}

func (s *Service) findCreateJob(jobID string) *wsm.JobReport {
	for _, job := range s.createJobIds {
		if job == jobID {
			return &wsm.JobReport{
				Id:         job,
				StatusCode: 200,
				Status:     wsm.JobReportStatusSUCCEEDED,
			}
		}
	}
	return nil
}

func (s *Service) removeWorkspace(workspaceID string) bool {
	for i, ws := range s.workspaces {
		if ws.Id.String() == workspaceID {
			s.workspaces = append(s.workspaces[:i], s.workspaces[i+1:]...)
			return true
		}
	}
	return false
}

func (s *Service) removeFolder(workspaceID, folderID string) bool {
	for _, ws := range s.workspaces {
		if ws.Id.String() == workspaceID {
			for j, folder := range ws.folders {
				if folder.Id.String() == folderID {
					ws.folders = append(ws.folders[:j], ws.folders[j+1:]...)
					return true
				}
			}
		}
	}
	return false
}

func remove(slice []string, s string) []string {
	for i, v := range slice {
		if v == s {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

func mergeProperties(oldProps, newProps []wsm.Property) []wsm.Property {
	// Build a map of oldProps for quick lookup
	oldMap := make(map[string]int)
	for i, p := range oldProps {
		oldMap[p.Key] = i
	}

	// Merge or add properties
	for _, newProp := range newProps {
		if idx, exists := oldMap[newProp.Key]; exists {
			oldProps[idx].Value = newProp.Value // Replace existing
		} else {
			oldProps = append(oldProps, newProp) // Add new
		}
	}

	return oldProps
}

func removeProperties(oldProps []wsm.Property, keys []string) []wsm.Property {
	// Build a map for quick lookup from keys
	keyMap := make(map[string]bool)
	for _, k := range keys {
		keyMap[k] = true
	}

	// Remove matching keys from oldProps
	var newProps []wsm.Property
	for _, p := range oldProps {
		if !keyMap[p.Key] {
			newProps = append(newProps, p)
		}
	}

	return newProps
}
