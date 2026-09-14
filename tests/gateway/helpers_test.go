package gateway_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func startServer(t *testing.T, f *fakeGateway) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(srv.Close)
	return srv.URL
}
