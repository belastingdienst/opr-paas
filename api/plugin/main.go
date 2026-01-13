package plugin

import "github.com/belastingdienst/opr-paas/v4/pkg/fields"

// Request represents the expected request payload for the plug-in generator.
//
// The ApplicationSetName identifies the target ApplicationSet in Argo CD.
// Input.Parameters is a map of user-provided parameters, where keys are
// strings and values are strings as well.
type Request struct {
	ApplicationSetName string `json:"applicationSetName"`
	Input              Input  `json:"input"`
}

// Input represents the input which is added to a Request
type Input struct {
	Parameters fields.ElementMap `json:"parameters"`
}

// Response represents the response payload returned by the plug-in generator.
//
// Output.Parameters is a slice of maps, where each map contains a set of
// key-value pairs representing generated parameters for the ApplicationSet.
type Response struct {
	Output Output `json:"output"`
}

// Output represents the output data which is added to a Response
type Output struct {
	Parameters []fields.ElementMap `json:"parameters"`
}
