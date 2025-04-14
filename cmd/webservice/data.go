/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package main

import "github.com/belastingdienst/opr-paas/api/v1alpha1"

// RestEncryptInput can be delivered to the API for encryption requests
type RestEncryptInput struct {
	PaasName string `json:"paas"`
	Secret   string `json:"secret"`
}

// RestEncryptResult is returned by the API for encryption requests
type RestEncryptResult struct {
	PaasName  string `json:"paas"`
	Encrypted string `json:"encrypted"`
	Valid     bool   `json:"valid"`
}

// RestCheckPaasInput can be delivered to the API for checkPaas requests
type RestCheckPaasInput struct {
	Paas v1alpha1.Paas `json:"paas"`
}

// RestCheckPaasResult is returned by the API for checkPaas requests
type RestCheckPaasResult struct {
	PaasName  string `json:"paas"`
	Decrypted bool   `json:"decrypted"`
	Error     string `json:"error"`
}

// RestGenerateInput is returned by the API for generate requests
type RestGenerateInput struct {
	APIKey string `json:"apiKey"`
}

// RestGenerateResult is returned by the API for generate requests
type RestGenerateResult struct {
	Private string `json:"private"`
	Public  string `json:"public"`
}
