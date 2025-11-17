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
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/belastingdienst/opr-paas/v3/api/plugin"
	"github.com/belastingdienst/opr-paas/v3/internal/logging"
	"github.com/belastingdienst/opr-paas/v3/pkg/fields"
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
	return &Handler{
		service:     service,
		bearerToken: bearerToken,
	}
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
	if authHeader != "Bearer "+h.bearerToken {
		logger.Error().Str("header", authHeader).Msg("invalid header")
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error().AnErr("error", err).Msg("invalid body")
		http.Error(w, fmt.Sprintf("read body error: %v", err), http.StatusBadRequest)
		return
	}

	var input plugin.Input
	if err = json.Unmarshal(body, &input); err != nil {
		logger.Error().Bytes("body", body).Msg("invalid json")
		http.Error(w, fmt.Sprintf("invalid json: %v", err), http.StatusBadRequest)
		return
	}

	result, err := h.service.Generate(ctx, input.Input.Parameters)
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
