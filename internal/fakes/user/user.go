// Package user provides a user-manager fake at the service API layer.
package user

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/verily-src/terraform-provider-workbench/internal/client"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/user"
)

// New creates a new user-manager Service struct.
func New() *Service {
	return &Service{
		groups:              make([]user.GroupDescriptionAndRoles, 0),
		groupGroupMemberMap: make(map[string][]user.GroupMember),
	}
}

// Service implements the APIs for a fake user-manager service.
type Service struct {
	sync.Mutex
	groups              []user.GroupDescriptionAndRoles
	groupGroupMemberMap map[string][]user.GroupMember // group name to group members map
}

func (s *Service) RegisterHTTP(e *echo.Echo) {
	g := e.Group("/api/user")
	// getGroup
	g.GET("/api/groups/v1/:groupName", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()

		groupName := c.Param("groupName")
		orgID := c.QueryParam("orgId")
		if orgID == "" {
			return jsonError(c, http.StatusBadRequest, "orgId is required")
		}
		uxid, isUfid := uxid(orgID)
		if !isUfid {
			if _, err := uuid.Parse(uxid); err != nil {
				return jsonError(c, http.StatusBadRequest, "invalid orgId %s", orgID)
			}
		}
		g, index := s.find(groupName, orgID)
		if index == -1 {
			return jsonError(c, http.StatusNotFound, "group %s in org %s not found", groupName, orgID)
		}
		return c.JSON(http.StatusOK, *g)
	})

	// create group
	g.POST("/api/organizations/v2/:orgId/groups", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()
		orgID := c.Param("orgId")

		var req user.CreateGroupRequest
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			return c.NoContent(http.StatusBadRequest)
		}

		groupName := req.GroupName
		if _, index := s.find(groupName, orgID); index >= 0 {
			return jsonError(c, http.StatusConflict, "group %s in org %s already exist", groupName, orgID)
		}
		// TODO(PHP-63134): validate org exists when we are faking org endpoints.
		orgUfid, orgUUID, err := getOrgUxids(orgID)
		if err != nil {
			return jsonError(c, http.StatusBadRequest, "invalid orgId %s", orgID)
		}

		var expirationDays int
		if req.ExpirationDays == nil {
			expirationDays = 0
		} else {
			expirationDays = *req.ExpirationDays
		}

		var syncGroup bool
		if req.SyncGroup == nil {
			syncGroup = true
		} else {
			syncGroup = *req.SyncGroup
		}

		now := time.Now()
		s.groups = append(s.groups, user.GroupDescriptionAndRoles{
			GroupAndRoles: user.GroupAndRoles{
				GroupEmail:   groupName + "@example.com",
				GroupName:    groupName,
				InternalName: &groupName,
				OrgId:        &orgUUID,
				OrgUfid:      &orgUfid,
			},
			GroupDescription: &user.GroupDescription{
				GroupEmail:             groupName + "@example.com",
				GroupName:              &groupName,
				InternalName:           &groupName,
				OrgId:                  orgUUID,
				OrgUfid:                orgUfid,
				CreatedBy:              "testuser",
				CreatedDate:            now,
				LastUpdatedBy:          "testuser",
				LastUpdatedDate:        now,
				Description:            req.Description,
				ExpirationDays:         expirationDays,
				ExpirationNotification: req.ExpirationNotification,
				RequireGrantReason:     req.RequireGrantReason,
				SyncGroup:              syncGroup,
			},
		})
		return c.JSON(http.StatusNoContent, nil)
	})

	// update group
	g.PATCH("/api/organizations/v2/:orgId/groups/:groupName", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()
		orgID := c.Param("orgId")
		groupName := c.Param("groupName")
		g, index := s.find(groupName, orgID)
		if index == -1 {
			return jsonError(c, http.StatusNotFound, "group %s in org %s not found", groupName, orgID)
		}
		var req user.UpdateGroupRequest
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			return jsonError(c, http.StatusBadRequest, "invalid request body: %s", err)
		}
		if req.Description == nil && req.Expiration == nil && req.ExpirationNotification == nil && req.RequireGrantReason == nil {
			return jsonError(c, http.StatusBadRequest, "at least one of description, expiration, expirationNotification or requireGrantReason is required")
		}
		if req.Description != nil {
			g.GroupDescription.Description = req.Description
		}
		if req.Expiration != nil {
			g.GroupDescription.ExpirationDays = int(*req.Expiration)
		}
		if req.ExpirationNotification != nil {
			g.GroupDescription.ExpirationNotification = req.ExpirationNotification
		}
		if req.RequireGrantReason != nil {
			g.GroupDescription.RequireGrantReason = req.RequireGrantReason
		}
		s.groups[index] = *g
		return c.JSON(http.StatusNoContent, nil)
	})

	// sync group
	g.PATCH("/api/organizations/v2/:orgId/groups/:groupName/sync", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()
		orgID := c.Param("orgId")
		groupName := c.Param("groupName")
		g, index := s.find(groupName, orgID)
		if index == -1 {
			return jsonError(c, http.StatusNotFound, "group %s in org %s not found", groupName, orgID)
		}
		g.GroupDescription.SyncGroup = true
		s.groups[index] = *g
		return c.JSON(http.StatusNoContent, nil)
	})

	// delete gorup
	g.DELETE("/api/organizations/v2/:orgId/groups/:groupName", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()

		orgID := c.Param("orgId")
		groupName := c.Param("groupName")
		if _, index := s.find(groupName, orgID); index == -1 {
			return jsonError(c, http.StatusNotFound, "group %s in org %s not found", groupName, orgID)
		}

		if ok := s.removeGroup(groupName, orgID); !ok {
			return jsonError(c, http.StatusNotFound, "group %s in org %s not found", groupName, orgID)
		}
		return c.JSON(http.StatusNoContent, nil)
	})

	// set group access
	g.POST("/api/groups/v1/:groupName/access", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()

		groupName := c.Param("groupName")
		orgID := c.QueryParam("orgId")

		if _, index := s.find(groupName, orgID); index == -1 {
			return jsonError(c, http.StatusNotFound, "group %s in org %s not found", groupName, orgID)
		}

		var req user.SetGroupAccessJSONRequestBody
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			return jsonError(c, http.StatusBadRequest, "invalid request body: %s", err)
		}

		role := req.Role
		if role != user.GroupRoleADMIN && role != user.GroupRoleMEMBER && role != user.GroupRoleREADER && role != user.GroupRoleSUPPORT {
			return jsonError(c, http.StatusBadRequest, "role must be one of ADMIN, MEMBER, READER, or SUPPORT")
		}

		orgAndGroup := orgID + "/" + groupName
		if req.Operation == user.GRANT {
			if gm, ok := s.groupGroupMemberMap[orgAndGroup]; ok {
				// If the group already has members, update them.
				for i, member := range gm {
					// Check if the principal matches the request principal, add the new role to the existing member if missing.
					if equal := reflect.DeepEqual(member.Principal, req.Principal); equal {
						member.Roles = appendRoleIfMissing(member.Roles, role)
						s.groupGroupMemberMap[orgAndGroup][i] = member
						return c.JSON(http.StatusNoContent, nil)
					}
				}
			}
			// If the group does not have a matching group members, create a new entry.
			groupMember := user.GroupMember{
				Principal: req.Principal,
				Roles:     []user.GroupRole{role},
			}
			s.groupGroupMemberMap[orgAndGroup] = append(s.groupGroupMemberMap[orgAndGroup], groupMember)
			return c.JSON(http.StatusNoContent, nil)
		}

		if req.Operation == user.REVOKE {
			if gm, ok := s.groupGroupMemberMap[orgAndGroup]; ok {
				// If the group has members, find the member to revoke.
				for i, member := range gm {
					if equal := reflect.DeepEqual(member.Principal, req.Principal); equal {
						// Remove the role from the member.
						member.Roles = removeRole(member.Roles, role)
						if len(member.Roles) == 0 {
							// If no roles left, remove the member.
							s.groupGroupMemberMap[orgAndGroup] = append(s.groupGroupMemberMap[orgAndGroup][:i], s.groupGroupMemberMap[orgAndGroup][i+1:]...)
						} else {
							s.groupGroupMemberMap[orgAndGroup][i] = member
						}
						return c.JSON(http.StatusNoContent, nil)
					}
				}
			}
			return c.JSON(http.StatusNoContent, nil)
		}
		return jsonError(c, http.StatusBadRequest, "invalid operation %s", req.Operation)
	})

	// list group membership
	g.GET("/api/groups/v1/:groupName/members", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()
		groupName := c.Param("groupName")
		orgID := c.QueryParam("orgId")

		if _, index := s.find(groupName, orgID); index == -1 {
			return jsonError(c, http.StatusNotFound, "group %s in org %s not found", groupName, orgID)
		}

		orgAndGroup := orgID + "/" + groupName
		if gm, ok := s.groupGroupMemberMap[orgAndGroup]; ok {
			return c.JSON(http.StatusOK, gm)
		}
		// If no members found, return an empty list.
		return c.JSON(http.StatusOK, user.GroupMemberList{})
	})

}

func jsonError(c echo.Context, code int, format string, a ...interface{}) error {
	return c.JSON(code, client.NewApiError(code, fmt.Sprintf(format, a...)))
}

func (s *Service) find(groupName string, orgID string) (*user.GroupDescriptionAndRoles, int) {
	uxid, isUfid := uxid(orgID)
	for i, g := range s.groups {
		if g.GroupDescription != nil && *g.GroupDescription.GroupName == groupName &&
			(isUfid && g.GroupDescription.OrgUfid == uxid) ||
			(!isUfid && g.GroupDescription.OrgId == uxid) {
			return &g, i
		}
	}
	return nil, -1
}

func (s *Service) removeGroup(groupName string, orgID string) bool {
	uxid, isUfid := uxid(orgID)
	for i, g := range s.groups {
		if g.GroupDescription != nil && *g.GroupDescription.GroupName == groupName &&
			(isUfid && g.GroupDescription.OrgUfid == uxid) ||
			(!isUfid && g.GroupDescription.OrgId == uxid) {
			s.groups = append(s.groups[:i], s.groups[i+1:]...)
			return true
		}
	}
	return false
}

func uxid(s string) (string, bool) {
	if strings.HasPrefix(s, "~") {
		return strings.TrimPrefix(s, "~"), true
	}
	return s, false
}

func getOrgUxids(orgID string) (string, string, error) {
	var orgUfid, orgUUID string
	if strings.HasPrefix(orgID, "~") {
		orgUfid = strings.TrimPrefix(orgID, "~")
		orgUUID = uuid.New().String()
	} else {
		if _, err := uuid.Parse(orgID); err != nil {
			return "", "", fmt.Errorf("invalid orgId %s", orgID)
		}
		orgUUID = orgID
		orgUfid = "a" + orgID
	}
	return orgUfid, orgUUID, nil
}

func appendRoleIfMissing(slice []user.GroupRole, elem user.GroupRole) []user.GroupRole {
	if slices.Contains(slice, elem) {
		return slice // Role already exists, return the original slice.
	}
	return append(slice, elem)
}

func removeRole(roles []user.GroupRole, role user.GroupRole) []user.GroupRole {
	for i, r := range roles {
		if r == role {
			return append(roles[:i], roles[i+1:]...)
		}
	}
	return roles
}
