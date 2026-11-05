package templating

import (
	"bytes"
	"maps"
	"text/template"

	"github.com/go-sprout/sprout"
	"github.com/go-sprout/sprout/registry/backward"
	"github.com/go-sprout/sprout/registry/checksum"
	"github.com/go-sprout/sprout/registry/conversion"
	"github.com/go-sprout/sprout/registry/encoding"
	"github.com/go-sprout/sprout/registry/filesystem"
	sproutmaps "github.com/go-sprout/sprout/registry/maps"
	"github.com/go-sprout/sprout/registry/network"
	"github.com/go-sprout/sprout/registry/numeric"
	"github.com/go-sprout/sprout/registry/random"
	"github.com/go-sprout/sprout/registry/reflect"
	"github.com/go-sprout/sprout/registry/regexp"
	"github.com/go-sprout/sprout/registry/semver"
	"github.com/go-sprout/sprout/registry/slices"
	"github.com/go-sprout/sprout/registry/std"
	"github.com/go-sprout/sprout/registry/strings"
	"github.com/go-sprout/sprout/registry/time"
	"github.com/go-sprout/sprout/registry/uniqueid"
	"github.com/goccy/go-yaml"

	"github.com/belastingdienst/opr-paas/v5/api"
	"github.com/belastingdienst/opr-paas/v5/api/v1alpha2"
	"github.com/belastingdienst/opr-paas/v5/pkg/fields"
)

// PaasUnion is an interface representing either a v1alpha1.Paas or a v1alpha2.Paas
type PaasUnion interface {
	// v1alpha1.Paas | v1alpha2.Paas
	v1alpha2.Paas
}

// Templater is a struct that can hold a Paas and a PaasConfig and can run go-templates using these as input
type Templater[P PaasUnion, C api.PaasConfig[S], S any] struct {
	Paas   P
	Config C
	// This should not be an exported value
	extraFuncs template.FuncMap
}

// NewTemplater returns an initialized Templater from a Paas and PaasConfig
func NewTemplater[P PaasUnion, C api.PaasConfig[S], S any](
	paas P,
	config C,
	extraFuncsMap ...template.FuncMap,
) Templater[P, C, S] {
	extraFuncs := template.FuncMap{}
	for _, fm := range extraFuncsMap {
		maps.Copy(extraFuncs, fm)
	}
	return Templater[P, C, S]{
		Paas:       paas,
		Config:     config,
		extraFuncs: extraFuncs,
	}
}

func (t Templater[P, C, S]) getSproutFuncs() (template.FuncMap, error) {
	var err error
	handler := sprout.New()
	// TODO: fail is currently deprecated. We need to check the community.
	// For using the fail function
	err = handler.AddRegistry(backward.NewRegistry())
	if err != nil {
		return nil, err
	}
	// Explicit list instead of all.RegistryGroup(): the env registry (env, expandEnv) is left out on purpose,
	// as it would expose the environment of the operator (e.g. ARGOCD_GENERATOR_TOKEN) to template output.
	err = handler.AddRegistries(
		checksum.NewRegistry(),
		conversion.NewRegistry(),
		encoding.NewRegistry(),
		filesystem.NewRegistry(), // path string manipulation only, no file access
		sproutmaps.NewRegistry(),
		network.NewRegistry(), // IP / CIDR parsing only, no network access
		numeric.NewRegistry(),
		random.NewRegistry(),
		reflect.NewRegistry(),
		regexp.NewRegistry(), //nolint:staticcheck // regex registry replaces this from sprout v1.2
		semver.NewRegistry(),
		slices.NewRegistry(),
		std.NewRegistry(),
		strings.NewRegistry(),
		time.NewRegistry(),
		uniqueid.NewRegistry(),
	)
	if err != nil {
		return nil, err
	}
	funcs := handler.Build()
	for _, name := range disallowedFuncs {
		delete(funcs, name)
	}
	return funcs, nil
}

// disallowedFuncs are functions from otherwise allowed registries that are removed, because they have side effects.
var disallowedFuncs = []string{
	"getHostByName", // backward registry: performs a DNS lookup from the operator
}

// Verify can verify a template (just parsing it, not running it against a Paas / PaasConfig)
func (t Templater[P, C, S]) Verify(name string, templatedText string) error {
	buf := new(bytes.Buffer)
	funcs, err := t.getSproutFuncs()
	if err != nil {
		return err
	}
	tmpl, err := template.New(name).Funcs(funcs).Funcs(t.extraFuncs).Parse(templatedText)
	if err != nil {
		return err
	}
	return tmpl.Execute(buf, t)
}

// TemplateToString can be used to parse a go-template and return a string value
func (t Templater[P, C, S]) TemplateToString(name string, templatedText string) (string, error) {
	buf := new(bytes.Buffer)
	funcs, err := t.getSproutFuncs()
	if err != nil {
		return "", err
	}
	tmpl, err := template.New(name).Funcs(funcs).Funcs(t.extraFuncs).Parse(templatedText)
	if err != nil {
		return "", err
	}
	err = tmpl.Execute(buf, t)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// TemplateToMap can be used to parse a go-template and try to parse as map or list.
// If it can be parsed, it will prefix map keys / list indexes by `name` and return the map.
// If it cannot be parsed as map / list, it will return a map with one key, value pair, where key = `name` and value
// is the result.
func (t Templater[P, C, S]) TemplateToMap(name string, templatedText string) (fields.ElementMap, error) {
	yamlData, templateErr := t.TemplateToString(name, templatedText)
	if templateErr != nil {
		return nil, templateErr
	}
	if myMap, err := yamlToMap([]byte(yamlData)); err == nil {
		return myMap.Prefix(name), nil
	}
	if myList, err := yamlToList([]byte(yamlData)); err == nil {
		return myList.AsElementMap().Prefix(name), nil
	}
	return fields.ElementMap{name: yamlData}, nil
}

// TemplateToStringMapStringMap can be used to parse a go-template and parse it as a string map of string maps.
// The most obvious use is the secret functions in the secret controller.
func (t Templater[P, C, S]) TemplateToStringMapStringMap(
	name string,
	templatedText string,
) (map[string]map[string]string, error) {
	buf := new(bytes.Buffer)
	funcs, err := t.getSproutFuncs()
	if err != nil {
		return nil, err
	}
	tmpl, err := template.New(name).Funcs(funcs).Funcs(t.extraFuncs).Parse(templatedText)
	if err != nil {
		return nil, err
	}
	err = tmpl.Execute(buf, t)
	if err != nil {
		return nil, err
	}
	myMap := map[string]map[string]string{}

	if marshalErr := yaml.Unmarshal(buf.Bytes(), &myMap); marshalErr != nil {
		return nil, marshalErr
	}
	return myMap, nil
}
