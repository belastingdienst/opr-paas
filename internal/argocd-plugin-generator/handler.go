/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/
/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package argocd_plugin_generator

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/belastingdienst/opr-paas/v5/api/plugin"
	"github.com/belastingdienst/opr-paas/v5/internal/logging"
	"github.com/belastingdienst/opr-paas/v5/pkg/fields"
)

// GeneratorService defines the contract for services that generate data
// for the plug-in generator. Implementations take arbitrary input parameters
// and an ApplicationSet name, then return a slice of key/value maps representing
// the generated output, or an error if generation fails.
type GeneratorService interface {
	Generate(ctx context.Context, params fields.ElementMap) ([]fields.ElementMap, error)
}

// Handler is the HTTP request handler for the plug-in generator.
//
// It validates incoming requests, enforces authentication using the
// configured bearer token, and delegates the core processing logic
// to the provided GeneratorService implementation.
type Handler struct {
	service     GeneratorService
	bearerToken string
}

// NewHandler creates a new Handler instance.
//
// The service parameter provides the generator's business logic,
// and bearerToken is used to authenticate incoming HTTP requests.
func NewHandler(service GeneratorService, bearerToken string) *Handler {
	if bearerToken == "" {
		_, logger := logging.GetLogComponent(context.Background(), logging.ApiComponent)
		logger.Fatal().Msg("Cannot use plugin generator without token")
	}
	return &Handler{
		service:     service,
		bearerToken: bearerToken,
	}
}

// maxRequestBodyBytes limits the size of a plugin request body. ArgoCD only sends a small
// JSON document with the ApplicationSet input parameters.
const maxRequestBodyBytes = 1 << 20 // 1 MiB

// validAuthHeader checks the Authorization header against the configured bearer token
// in constant time, so the comparison does not leak information about the token through timing.
func (h *Handler) validAuthHeader(authHeader string) bool {
	expected := []byte("Bearer " + h.bearerToken)
	return subtle.ConstantTimeCompare([]byte(authHeader), expected) == 1
}

// ServeHTTP implements the http.Handler interface.
//
// It reads and validates the incoming request, delegates to the Service
// for processing, and encodes the output as JSON. In case of errors,
// an appropriate HTTP status code and error message are returned.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, logger := logging.SetPluginLogger(r.Context(), r)

	if r.Method != http.MethodPost || r.URL.Path != "/api/v1/getparams.execute" {
		logger.Error().Msg("invalid request method or path")
		http.NotFound(w, r)
		return
	}

	authHeader := r.Header.Get("Authorization")
	if !h.validAuthHeader(authHeader) {
		// Never log the (rejected) header value: it may still be a valid credential elsewhere.
		logger.Error().
			Bool("header_present", authHeader != "").
			Bool("bearer_scheme", strings.HasPrefix(authHeader, "Bearer ")).
			Msg("authentication failed")
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxRequestBodyBytes))
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			logger.Error().Int64("limit", maxBytesErr.Limit).Msg("request body too large")
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		logger.Error().AnErr("error", err).Msg("invalid body")
		http.Error(w, fmt.Sprintf("read body error: %v", err), http.StatusBadRequest)
		return
	}

	var request plugin.Request
	if err = json.Unmarshal(body, &request); err != nil {
		// Do not log the body itself: it is unbounded, caller-controlled input.
		logger.Error().Int("body_length", len(body)).AnErr("error", err).Msg("invalid json")
		http.Error(w, fmt.Sprintf("invalid json: %v", err), http.StatusBadRequest)
		return
	}

	result, err := h.service.Generate(ctx, request.Input.Parameters)
	if err != nil {
		logger.Error().AnErr("error", err).Msg("generation error")
		http.Error(w, fmt.Sprintf("generation error: %v", err), http.StatusInternalServerError)
		return
	}

	if result == nil {
		logger.Debug().Msg("generate returns nil")
		result = []fields.ElementMap{}
	}
	logger.Debug().Int("num_capabilities", len(result)).Msg("generate succeeded")

	response := plugin.Response{}
	response.Output.Parameters = result

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		logger.Error().AnErr("error", err).Msg("json encoder failure")
		return
	}
	logger.Debug().Msg("OK")
}
