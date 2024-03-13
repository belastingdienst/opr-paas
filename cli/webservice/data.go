/*
Copyright 2023, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package main

type RestEncryptInput struct {
	PaasName string `json:"paas"`
	Secret   string `json:"secret"`
}

type RestEncryptResult struct {
	PaasName  string `json:"paas"`
	Encrypted string `json:"encrypted"`
	Valid     bool   `json:"valid"`
}

type RestGenerateInput struct {
	ApiKey string `json:"apiKey"`
}

type RestGenerateResult struct {
	Private string `json:"private"`
	Public  string `json:"public"`
}
