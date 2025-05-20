package api

type PaasConfig[S any] interface {
	GetSpec() S
}

type ConfigCapabilities interface{}
