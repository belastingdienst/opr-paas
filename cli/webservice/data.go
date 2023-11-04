package main

type RestInput struct {
	PaasName string `json:"paas"`
	Secret   string `json:"secret"`
}

type RestVersionResult struct {
	Version string `json:"version"`
}

type RestEncryptResult struct {
	PaasName  string `json:"paas"`
	Encrypted string `json:"encrypted"`
}

type RestGenerateResult struct {
	Private string `json:"private"`
	Public  string `json:"public"`
}
