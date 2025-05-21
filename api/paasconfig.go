package api

// PaasConfig is the generic interface for a Paas config, basically mandating
// that any version PaasConfig has a GetSpec method which returns a spec.
type PaasConfig[S any] interface {
	GetSpec() S
}

// ConfigCapabilities is a generic interface needed to allow the generic PaasConfig
// interface.
type ConfigCapabilities interface{}
