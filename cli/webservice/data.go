package main

type RestEncryptInput struct {
	PaasName string `json:"paas"`
	Secret   string `json:"secret"`
}

type RestEncryptResult struct {
	PaasName  string `json:"paas"`
	Encrypted string `json:"encrypted"`
}

type RestGenerateInput struct {
	ApiKey string `json:"apiKey"`
}

type RestGenerateResult struct {
	Private string `json:"private"`
	Public  string `json:"public"`
}
