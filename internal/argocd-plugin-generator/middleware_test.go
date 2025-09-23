/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package argocd_plugin_generator

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestWithMetrics_IncrementsCounter(t *testing.T) {
	// Reset metrics before test
	PluginGeneratorRequestTotal.Reset()

	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantStatus int
	}{
		{
			name: "200 OK",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := io.WriteString(w, "ok")
				if err != nil {
					return
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "404 Not Found",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.NotFound(w, r)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "500 Internal Server Error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "fail", http.StatusInternalServerError)
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)

			handler := withMetrics(tt.handler)
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}

			// Verify counter increment
			got := testutil.ToFloat64(PluginGeneratorRequestTotal.WithLabelValues(strconv.Itoa(tt.wantStatus)))
			if got != 1 {
				t.Errorf("expected counter 1 for status %d, got %v", tt.wantStatus, got)
			}

			// Reset for next run
			PluginGeneratorRequestTotal.Reset()
		})
	}
}
