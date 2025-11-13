package plugin

import "github.com/belastingdienst/opr-paas/v3/pkg/fields"

// Input represents the expected request payload for the plug-in generator.
//
// The ApplicationSetName identifies the target ApplicationSet in Argo CD.
// Input.Parameters is a map of user-provided parameters, where keys are
// strings and values are strings as well.
type Input struct {
	ApplicationSetName string `json:"applicationSetName"`
	Input              struct {
		Parameters fields.ElementMap `json:"parameters"`
	} `json:"input"`
}

// Response represents the response payload returned by the plug-in generator.
//
// Output.Parameters is a slice of maps, where each map contains a set of
// key-value pairs representing generated parameters for the ApplicationSet.
type Response struct {
	Output struct {
		Parameters []fields.ElementMap `json:"parameters"`
	} `json:"output"`
}
