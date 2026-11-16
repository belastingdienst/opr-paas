/*
Copyright 2026, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package argocd_plugin_generator

import (
	"bytes"
	"sync"

	"github.com/belastingdienst/opr-paas/v5/internal/logging"
	. "github.com/onsi/ginkgo/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// syncBuffer is a bytes.Buffer that is safe for concurrent use (e.g. by an httptest server goroutine).
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// captureDebugLogs redirects all zerolog output (global logger and context loggers) to a buffer
// with debug logging enabled for all components. Everything is restored when the current spec ends.
// Use it to assert that sensitive values never end up in log output.
func captureDebugLogs() *syncBuffer {
	buf := &syncBuffer{}

	orgLogger := log.Logger
	orgDefaultContextLogger := zerolog.DefaultContextLogger
	DeferCleanup(func() {
		log.Logger = orgLogger
		zerolog.DefaultContextLogger = orgDefaultContextLogger
		logging.SetStaticLoggingConfig(false, logging.Components{})
	})

	log.Logger = zerolog.New(buf).Level(zerolog.DebugLevel)
	captured := log.Logger
	zerolog.DefaultContextLogger = &captured
	logging.SetStaticLoggingConfig(true, logging.Components{})
	return buf
}
