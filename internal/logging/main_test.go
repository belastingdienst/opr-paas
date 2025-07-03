package logging

import (
	"context"
	"fmt"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	controllerruntime "sigs.k8s.io/controller-runtime"

	"github.com/belastingdienst/opr-paas/v3/api/v1alpha1"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	paasName = "my-paas"
)

type logSink struct {
	logs []string
}

func (l *logSink) Write(p []byte) (n int, err error) {
	l.logs = append(l.logs, string(p))
	return len(p), nil
}

func (l *logSink) Index(i int) string {
	if len(l.logs) >= i {
		return l.logs[i]
	}
	return ""
}

func TestSetControllerLogger(t *testing.T) {
	ctx := context.TODO()
	obj := &v1alpha1.Paas{
		ObjectMeta: metav1.ObjectMeta{
			Name: paasName,
		},
		Spec: v1alpha1.PaasSpec{},
	}
	runtimeSchema := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(runtimeSchema)
	req := controllerruntime.Request{}

	output := &logSink{}
	log.Logger = log.Output(output)
	_, logger := SetControllerLogger(ctx, obj, runtimeSchema, req)
	require.NotNil(t, logger, "SetControllerLogger should return a logger")

	logger.Log().Msg("some controller log")
	require.Len(t, output.logs, 2, "There should be 2 item in logs")
	logLine := output.Index(1)
	expectedPrefix := `{"controller":`
	assert.True(t, strings.HasPrefix(logLine, expectedPrefix), "logline should begin with `%s`", expectedPrefix)
	assert.Contains(t, logLine, `"Group":"cpet.belastingdienst.nl"`)
	assert.Contains(t, logLine, `"Kind":"Paas"`)
	assert.Contains(t, logLine, `"message":"some controller log"`)
	assert.Contains(t, logLine, `"object":{"Namespace":"","Name":""}`)
}

func TestSetControllerLoggerUnknownGVK(t *testing.T) {
	ctx := context.Background()
	runtimeSchema := runtime.NewScheme()
	obj := &v1alpha1.Paas{}
	output := &logSink{}
	log.Logger = log.Output(output)
	_, logger := SetControllerLogger(ctx, obj, runtimeSchema, controllerruntime.Request{})

	assert.NotNil(t, logger)
	assert.Contains(t, output.Index(0), "no kind is registered for the type v1alpha1.Paas")
}

func TestSetWebhookLogger(t *testing.T) {
	ctx := context.TODO()
	obj := &v1alpha1.Paas{
		ObjectMeta: metav1.ObjectMeta{
			Name: paasName,
		},
		Spec: v1alpha1.PaasSpec{},
	}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Kind:    "Paas",
		Version: "v1alpha1",
		Group:   "cpet.belastingdienst.nl",
	})

	output := &logSink{}
	log.Logger = log.Output(output)
	_, logger := SetWebhookLogger(ctx, obj)
	require.NotNil(t, logger, "SetWebhookLogger should return a logger")

	logger.Log().Msg("some webhook log")
	require.Len(t, output.logs, 2, "There should be 1 item in logs")
	logLine := output.Index(1)
	expectedPrefix := `{"webhook":`
	assert.True(t, strings.HasPrefix(logLine, expectedPrefix), "logline should begin with `%s`", expectedPrefix)
	assert.Contains(t, logLine, `"Group":"cpet.belastingdienst.nl"`)
	assert.Contains(t, logLine, `"Kind":"Paas"`)
	assert.Contains(t, logLine, `"message":"some webhook log"`)
	assert.Contains(t, logLine, fmt.Sprintf(`"object":{"name":"%s","namespace":""}`, paasName))
}

func TestSetComponentDebug(t *testing.T) {
	ResetComponentDebug()
	dbgCmp := []string{"comp1", "comp2"}
	defer ResetComponentDebug()
	SetComponentDebug(dbgCmp)
	require.Len(t, debugComponents, 2, "there should be 2 components in debug")
	for _, c := range dbgCmp {
		_, exists := debugComponents[c]
		assert.True(t, exists, "%s should be in debugComponents")
	}
	_, exists := debugComponents["comp3"]
	assert.False(t, exists, "comp3 should not be in debugComponents")
}

func TestSetLogComponent(t *testing.T) {
	ResetComponentDebug()
	obj := &v1alpha1.Paas{
		ObjectMeta: metav1.ObjectMeta{
			Name: paasName,
		},
		Spec: v1alpha1.PaasSpec{},
	}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Kind:    "Paas",
		Version: "v1alpha1",
		Group:   "cpet.belastingdienst.nl",
	})

	debugCmp := "comp1"
	SetComponentDebug([]string{debugCmp})
	for _, comp := range []string{"comp1", "comp2"} {
		ctx := context.Background()
		output := &logSink{}
		log.Logger = log.Output(output)
		ctx, _ = SetWebhookLogger(ctx, obj)
		_, logger := GetLogComponent(ctx, comp)
		require.NotNil(t, logger, "GetLogComponent should return a logger")
		expected := zerolog.InfoLevel
		if debugCmp == comp {
			expected = zerolog.DebugLevel
		}
		assert.Equal(t, expected, logger.GetLevel(), "component %s should be %d", comp, expected)
	}
}
