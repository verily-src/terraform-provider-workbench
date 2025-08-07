// Package fakes provides testing fakes for other services.
package fakes

import (
	"net"
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"
)

type httpService interface {
	RegisterHTTP(e *echo.Echo)
}

// Use enables the specified fakes in tests.  Any errors during initialization
// will cause the test to fail.
// The server address is returned.
func Use(t *testing.T, fakes ...httpService) string {
	e := echo.New()

	for _, fake := range fakes {
		fake.RegisterHTTP(e)
	}
	s := newServer(e)
	return s.start(t)
}

type server struct {
	*http.Server
}

func newServer(handler http.Handler) *server {
	return &server{
		&http.Server{Handler: handler},
	}
}

func (s *server) start(t *testing.T) string {
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	addr := lis.Addr().String()
	t.Logf("Fakes server running at %v", addr)

	go func() {
		if err := s.Serve(lis); err != nil && err != http.ErrServerClosed {
			t.Logf("Serving error: %v", err)
		}
	}()
	t.Cleanup(func() { s.Close() })
	return addr
}
