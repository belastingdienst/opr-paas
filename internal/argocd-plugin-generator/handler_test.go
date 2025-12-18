/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package argocd_plugin_generator

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/belastingdienst/opr-paas/v4/api/plugin"
	"github.com/belastingdienst/opr-paas/v4/pkg/fields"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// mockGeneratorService implements GeneratorService for tests
type mockGeneratorService struct {
	generateFunc func(params fields.ElementMap) ([]fields.ElementMap, error)
}

func (m *mockGeneratorService) Generate(ctx context.Context, params fields.ElementMap) (
	[]fields.ElementMap, error,
) {
	return m.generateFunc(params)
}

var _ = Describe("Handler", func() {
	var (
		mockService *mockGeneratorService
		bearerToken string
		handler     *Handler
		server      *httptest.Server
		httpClient  *http.Client
	)

	BeforeEach(func() {
		bearerToken = "supersecrettoken"
		mockService = &mockGeneratorService{}
		handler = NewHandler(mockService, bearerToken)
		server = httptest.NewServer(handler)
		httpClient = server.Client()
	})

	AfterEach(func() {
		server.Close()
	})

	Context("ServeHTTP", func() {
		It("returns 404 for wrong method", func() {
			req, _ := http.NewRequest(http.MethodGet, server.URL+"/api/v1/getparams.execute", nil)
			resp, err := httpClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		})

		It("returns 404 for wrong path", func() {
			req, _ := http.NewRequest(http.MethodPost, server.URL+"/wrongpath", nil)
			resp, err := httpClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		})

		It("returns 403 if bearer token is missing or incorrect", func() {
			req, _ := http.NewRequest(http.MethodPost, server.URL+"/api/v1/getparams.execute", nil)
			resp, err := httpClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
		})

		It("returns 400 if body cannot be read", func() {
			mockService.generateFunc = func(params fields.ElementMap) (
				[]fields.ElementMap, error,
			) {
				return nil, nil
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/getparams.execute", errReader(0))
			req.Header.Set("Authorization", "Bearer "+bearerToken)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusBadRequest))
			Expect(rr.Body.String()).To(ContainSubstring("read body error"))
		})

		It("returns 400 if body contains invalid JSON", func() {
			req, _ := http.NewRequest(http.MethodPost, server.URL+"/api/v1/getparams.execute",
				bytes.NewBufferString("{invalid json"))
			req.Header.Set("Authorization", "Bearer "+bearerToken)
			resp, err := httpClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
			body, _ := io.ReadAll(resp.Body)
			Expect(string(body)).To(ContainSubstring("invalid json"))
		})

		It("returns 500 if Service.Generate returns an error", func() {
			mockService.generateFunc = func(params fields.ElementMap) (
				[]fields.ElementMap, error,
			) {
				return nil, errors.New("generation failed")
			}

			payload := fields.ElementMap{
				"applicationSetName": "appset1",
				"input": fields.ElementMap{
					"parameters": fields.ElementMap{"foo": "bar"},
				},
			}
			body, _ := json.Marshal(payload)

			req, _ := http.NewRequest(http.MethodPost, server.URL+"/api/v1/getparams.execute", bytes.NewBuffer(body))
			req.Header.Set("Authorization", "Bearer "+bearerToken)
			resp, err := httpClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
			respBody, _ := io.ReadAll(resp.Body)
			Expect(string(respBody)).To(ContainSubstring("generation error"))
		})

		It("returns 200 and JSON when Generate succeeds", func() {
			expectedResult := []fields.ElementMap{
				{"key1": "value1"},
				{"key2": "value2"},
			}
			mockService.generateFunc = func(params fields.ElementMap) (
				[]fields.ElementMap, error,
			) {
				Expect(params).To(HaveKeyWithValue("foo", "bar"))
				return expectedResult, nil
			}

			payload := fields.ElementMap{
				"applicationSetName": "appset1",
				"input": fields.ElementMap{
					"parameters": fields.ElementMap{"foo": "bar"},
				},
			}
			body, _ := json.Marshal(payload)

			req, _ := http.NewRequest(http.MethodPost, server.URL+"/api/v1/getparams.execute", bytes.NewBuffer(body))
			req.Header.Set("Authorization", "Bearer "+bearerToken)
			resp, err := httpClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			var jsonResp plugin.Response
			Expect(json.NewDecoder(resp.Body).Decode(&jsonResp)).To(Succeed())
			Expect(jsonResp.Output.Parameters).To(Equal(
				[]fields.ElementMap{{"key1": "value1"}, {"key2": "value2"}},
			))
		})
	})
})

// errReader simulates a body read error
type errReader int

func (errReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("read error")
}
