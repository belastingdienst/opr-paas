package logging

import "strings"

// Components is a map that holds components and their Debug state (false is InfoLevel, True is DebugLevel)
type Components map[Component]bool

// Component is a custom type, so that we can use it as an ENUM
type Component int

const (
	// RuntimeComponent represents a logging component for the runtime controller
	// (Note: As the runtime logger is only fetched once, changing debuglevel with PaasConfix has no effect.)
	RuntimeComponent Component = iota
	// ApiComponent represents a logging component for the runtime controller.
	// Note: As the runtime logger is only fetched once, changing debuglevel with PaasConfix has no effect.
	ApiComponent Component = iota

	// WebhookPaasConfigComponentV1 represents a logging component for the v1alpha1 code for the PaasConfig webhook
	WebhookPaasConfigComponentV1 Component = iota
	// WebhookPaasComponentV1 represents a logging component for the v1alpha1 code for the Paas webhook
	WebhookPaasComponentV1 Component = iota
	// WebhookPaasNSComponentV1 represents a logging component for the v1alpha1 code for the PaasNS webhook
	WebhookPaasNSComponentV1 Component = iota
	// WebhookUtilsComponentV1 represents a logging component for the v1alpha1 utils code
	WebhookUtilsComponentV1 Component = iota
	// WebhookPaasConfigComponentV2 represents a logging component for the v1alpha2 code for the PaasConfig webhook
	WebhookPaasConfigComponentV2 Component = iota
	// WebhookPaasComponentV2 represents a logging component for the v1alpha2 code for the Paas webhook
	WebhookPaasComponentV2 Component = iota
	// WebhookPaasNSComponentV2 represents a logging component for the v1alpha2 code for the PaasNS webhook
	WebhookPaasNSComponentV2 Component = iota
	// WebhookUtilsComponentV2 represents a logging component for the v1alpha2 utils code
	WebhookUtilsComponentV2 Component = iota

	// PluginGeneratorComponent represents a logging component used by the argocd plugin generator
	PluginGeneratorComponent Component = iota

	// ConfigComponent represents a logging component used by the config cache
	ConfigComponent Component = iota

	// ControllerCapabilitiesComponent represents a logging component used by the capabilities controller
	ControllerCapabilitiesComponent Component = iota
	// ControllerClusterQuotaComponent represents a logging component used by the cluster quota controller
	ControllerClusterQuotaComponent Component = iota
	// ControllerClusterRoleBindingsComponent represents a logging component used by the cluster rolebindings controller
	ControllerClusterRoleBindingsComponent Component = iota
	// ControllerGroupComponent represents a logging component used by the group controller
	ControllerGroupComponent Component = iota
	// ControllerNamespaceComponent represents a logging component used by the namespace controller
	ControllerNamespaceComponent Component = iota
	// ControllerPaasComponent represents a logging component used by the paas controller
	ControllerPaasComponent Component = iota
	// ControllerPaasConfigComponent represents a logging component used by the paasConfig controller
	ControllerPaasConfigComponent Component = iota
	// ControllerRoleBindingComponent represents a logging component used by the role binding controller
	ControllerRoleBindingComponent Component = iota
	// ControllerSecretComponent represents a logging component used by the secret controller
	ControllerSecretComponent Component = iota

	// UnknownComponent represents a logging component with unknown origin
	UnknownComponent Component = iota
	// TestComponent represents a logging component only used in unittests
	TestComponent Component = iota
)

var (
	componentConverter = map[string]Component{
		"runtime": RuntimeComponent,
		"api":     ApiComponent,

		"paasconfig_webhook_v1": WebhookPaasConfigComponentV1,
		"paas_webhook_v1":       WebhookPaasComponentV1,
		"paasns_webhook_v1":     WebhookPaasNSComponentV1,
		"utils_webhook_v1":      WebhookUtilsComponentV1,
		"paasconfig_webhook_v2": WebhookPaasConfigComponentV2,
		"paas_webhook_v2":       WebhookPaasComponentV2,
		"paasns_webhook_v2":     WebhookPaasNSComponentV2,
		"utils_webhook_v2":      WebhookUtilsComponentV2,

		"capabilities_controller":         ControllerCapabilitiesComponent,
		"cluster_quota_controller":        ControllerClusterQuotaComponent,
		"cluster_role_binding_controller": ControllerClusterRoleBindingsComponent,
		"group_controller":                ControllerGroupComponent,
		"namespace_controller":            ControllerNamespaceComponent,
		"paas_controller":                 ControllerPaasComponent,
		"paas_config_controller":          ControllerPaasConfigComponent,
		"rolebinding_controller":          ControllerRoleBindingComponent,
		"secret_controller":               ControllerSecretComponent,

		"plugin_generator": PluginGeneratorComponent,
		"config_watcher":   ConfigComponent,

		"undefined_component": UnknownComponent,
		"unittest_component":  TestComponent,
	}
	reverseComponentConverter map[Component]string
)

func componentToString(component Component) string {
	revConverter := map[Component]string{}
	if reverseComponentConverter == nil {
		for s, comp := range componentConverter {
			revConverter[comp] = s
		}
	}
	if s, exists := revConverter[component]; exists {
		return s
	}
	return "undefined_component"
}

// NewComponentsFromString takes a comma separated string (as used in command arguments) and converts it into a
// Components object
func NewComponentsFromString(commaSeparated string) Components {
	components := Components{}
	for _, compName := range strings.Split(commaSeparated, ",") {
		if component, exists := componentConverter[compName]; exists {
			components[component] = true
		} else {
			components[UnknownComponent] = true
		}
	}
	return components
}

// NewComponentsFromStringMap takes a string map (as used in PaasConfig) and converts it into a Components object
func NewComponentsFromStringMap(enabledComponents map[string]bool) Components {
	components := Components{}
	for compName, state := range enabledComponents {
		if component, exists := componentConverter[compName]; exists {
			components[component] = state
		} else {
			components[UnknownComponent] = state
		}
	}
	return components
}
