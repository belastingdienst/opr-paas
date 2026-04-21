package logging

import (
	"context"
	"fmt"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	controllerruntime "sigs.k8s.io/controller-runtime"

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
	obj := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: paasName,
		},
	}
	runtimeSchema := runtime.NewScheme()
	_ = corev1.AddToScheme(runtimeSchema)
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
	assert.Contains(t, logLine, `"Group":""`)
	assert.Contains(t, logLine, `"Kind":"Namespace"`)
	assert.Contains(t, logLine, `"message":"some controller log"`)
	assert.Contains(t, logLine, `"object":{"Namespace":"","Name":""}`)
}

func TestSetControllerLoggerUnknownGVK(t *testing.T) {
	ctx := context.Background()
	runtimeSchema := runtime.NewScheme()
	obj := &corev1.Namespace{}
	output := &logSink{}
	log.Logger = log.Output(output)
	_, logger := SetControllerLogger(ctx, obj, runtimeSchema, controllerruntime.Request{})

	assert.NotNil(t, logger)
	assert.Contains(t, output.Index(0), "no kind is registered for the type v1.Namespace")
}

func TestSetWebhookLogger(t *testing.T) {
	ctx := context.TODO()
	obj := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: paasName,
		},
	}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Kind: "Paas",
		// Setting this to v1alpha0 to make it very distinct from whatever we actually use
		Version: "v1alpha0",
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

func TestDebuggingStatic(t *testing.T) {
	const comp1 = TestComponent
	SetDynamicLoggingConfig(false, nil)
	ctx := context.TODO()
	// debug false
	SetStaticLoggingConfig(false, nil)
	_, noDebugLogger := GetLogComponent(ctx, comp1)
	assert.Equal(t, zerolog.InfoLevel, noDebugLogger.GetLevel())
	// debug true
	SetStaticLoggingConfig(true, nil)
	_, allDebugLogger := GetLogComponent(ctx, comp1)
	assert.Equal(t, zerolog.DebugLevel, allDebugLogger.GetLevel())
	// debug component
	SetStaticLoggingConfig(false, Components{comp1: true})
	_, componentDebugLogger := GetLogComponent(ctx, comp1)
	assert.Equal(t, zerolog.DebugLevel, componentDebugLogger.GetLevel())
}

func TestDebuggingConfig(t *testing.T) {
	const comp1 = TestComponent
	SetStaticLoggingConfig(false, nil)
	ctx := context.TODO()
	// debug false
	SetDynamicLoggingConfig(false, nil)
	_, noDebugLogger := GetLogComponent(ctx, comp1)
	assert.Equal(t, zerolog.InfoLevel, noDebugLogger.GetLevel())
	// debug true
	SetDynamicLoggingConfig(true, nil)
	_, allDebugLogger := GetLogComponent(ctx, comp1)
	assert.Equal(t, zerolog.DebugLevel, allDebugLogger.GetLevel())
	// debug component on
	SetDynamicLoggingConfig(false, map[Component]bool{comp1: true})
	_, componentDebugLogger := GetLogComponent(ctx, comp1)
	assert.Equal(t, zerolog.DebugLevel, componentDebugLogger.GetLevel())

	// debug component off
	SetStaticLoggingConfig(true, Components{comp1: true})
	SetDynamicLoggingConfig(true, map[Component]bool{comp1: false})
	_, componentNoDebugLogger := GetLogComponent(ctx, comp1)
	assert.Equal(t, zerolog.InfoLevel, componentNoDebugLogger.GetLevel())
}
