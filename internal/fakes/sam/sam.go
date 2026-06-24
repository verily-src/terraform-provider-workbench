package sam

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/verily-src/terraform-provider-workbench/internal/client"
	"github.com/verily-src/terraform-provider-workbench/internal/openapi/sam"
)

func New() *Service {
	return &Service{
		perimeters: make(map[string]*perimeterState),
	}
}

type Service struct {
	sync.Mutex
	perimeters map[string]*perimeterState
}

type perimeterState struct {
	policies  map[string][]string
	synced    bool
	syncEmail string
}

func (s *Service) RegisterHTTP(e *echo.Echo) {
	g := e.Group("/api/sam")

	g.POST("/api/resources/v2/perimeter", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()

		var req sam.CreateResourceRequestV2
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			return jsonError(c, http.StatusBadRequest, "invalid request body: %s", err)
		}

		if _, exists := s.perimeters[req.ResourceId]; exists {
			return jsonError(c, http.StatusConflict, "perimeter %s already exists", req.ResourceId)
		}

		policies := make(map[string][]string)
		for name, policy := range req.Policies {
			policies[name] = policy.MemberEmails
		}

		s.perimeters[req.ResourceId] = &perimeterState{
			policies: policies,
		}

		return c.NoContent(http.StatusNoContent)
	})

	g.GET("/api/resources/v2/perimeter/:resourceId/policies", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()

		resourceId := c.Param("resourceId")
		p, ok := s.perimeters[resourceId]
		if !ok {
			return jsonError(c, http.StatusNotFound, "perimeter %s not found", resourceId)
		}

		var entries []sam.AccessPolicyResponseEntryV2
		for name, emails := range p.policies {
			var roles []string
			switch name {
			case "owner":
				roles = []string{"owner"}
			case "user":
				roles = []string{"user"}
			}
			entry := sam.AccessPolicyResponseEntryV2{
				PolicyName: name,
				Policy: sam.AccessPolicyMembershipV2{
					MemberEmails: emails,
					Roles:        roles,
					Actions:      []string{},
				},
			}
			if name == "user" && p.synced {
				entry.Email = p.syncEmail
			}
			entries = append(entries, entry)
		}

		return c.JSON(http.StatusOK, entries)
	})

	g.PUT("/api/resources/v2/perimeter/:resourceId/policies/:policyName", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()

		resourceId := c.Param("resourceId")
		policyName := c.Param("policyName")

		p, ok := s.perimeters[resourceId]
		if !ok {
			return jsonError(c, http.StatusNotFound, "perimeter %s not found", resourceId)
		}

		var req sam.AccessPolicyMembershipRequest
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			return jsonError(c, http.StatusBadRequest, "invalid request body: %s", err)
		}

		p.policies[policyName] = req.MemberEmails
		return c.NoContent(http.StatusCreated)
	})

	g.DELETE("/api/resources/v2/perimeter/:resourceId", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()

		resourceId := c.Param("resourceId")
		if _, ok := s.perimeters[resourceId]; !ok {
			return jsonError(c, http.StatusNotFound, "perimeter %s not found", resourceId)
		}

		delete(s.perimeters, resourceId)
		return c.NoContent(http.StatusNoContent)
	})

	g.POST("/api/google/v1/resource/perimeter/:resourceId/user/sync", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()

		resourceId := c.Param("resourceId")
		p, ok := s.perimeters[resourceId]
		if !ok {
			return jsonError(c, http.StatusNotFound, "perimeter %s not found", resourceId)
		}

		if !p.synced {
			p.synced = true
			p.syncEmail = fmt.Sprintf("policy-%s@verily-bvdp.com", uuid.New().String())
		}

		return c.JSON(http.StatusOK, sam.SyncReport{})
	})

	g.GET("/api/google/v1/resource/perimeter/:resourceId/user/sync", func(c echo.Context) error {
		s.Lock()
		defer s.Unlock()

		resourceId := c.Param("resourceId")
		p, ok := s.perimeters[resourceId]
		if !ok {
			return jsonError(c, http.StatusNotFound, "perimeter %s not found", resourceId)
		}

		if !p.synced {
			return c.NoContent(http.StatusNoContent)
		}

		return c.JSON(http.StatusOK, sam.SyncStatus{
			LastSyncDate: "2024-01-01T00:00:00Z",
			Email:        p.syncEmail,
		})
	})
}

func jsonError(c echo.Context, code int, format string, a ...interface{}) error {
	return c.JSON(code, client.NewApiError(code, fmt.Sprintf(format, a...)))
}
