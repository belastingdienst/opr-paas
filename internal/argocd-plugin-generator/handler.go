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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// GeneratorService defines the contract for services that generate data
// for the plug-in generator. Implementations take arbitrary input parameters
// and an ApplicationSet name, then return a slice of key/value maps representing
// the generated output, or an error if generation fails.
type GeneratorService interface {
	Generate(params map[string]interface{}, appSetName string) ([]map[string]interface{}, error)
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
	if r.Method != http.MethodPost || r.URL.Path != "/api/v1/getparams.execute" {
		http.NotFound(w, r)
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader != "Bearer "+h.bearerToken {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("read body error: %v", err), http.StatusBadRequest)
		return
	}

	var input PluginInput
	if err = json.Unmarshal(body, &input); err != nil {
		http.Error(w, fmt.Sprintf("invalid json: %v", err), http.StatusBadRequest)
		return
	}

	result, err := h.service.Generate(input.Input.Parameters, input.ApplicationSetName)
	if err != nil {
		http.Error(w, fmt.Sprintf("generation error: %v", err), http.StatusInternalServerError)
		return
	}

	if result == nil {
		result = []map[string]interface{}{}
	}

	response := PluginResponse{}
	response.Output.Parameters = result

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		return
	}
}

// PluginInput represents the expected request payload for the plug-in generator.
//
// The ApplicationSetName identifies the target ApplicationSet in Argo CD.
// Input.Parameters is a map of user-provided parameters, where keys are
// strings and values are strings as well.
type PluginInput struct {
	ApplicationSetName string `json:"applicationSetName"`
	Input              struct {
		Parameters map[string]interface{} `json:"parameters"`
	} `json:"input"`
}

// PluginResponse represents the response payload returned by the plug-in generator.
//
// Output.Parameters is a slice of maps, where each map contains a set of
// key-value pairs representing generated parameters for the ApplicationSet.
type PluginResponse struct {
	Output struct {
		Parameters []map[string]interface{} `json:"parameters"`
	} `json:"output"`
}
