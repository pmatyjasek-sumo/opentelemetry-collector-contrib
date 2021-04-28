// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sapmexporter

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jaegertracing/jaeger/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/translator/trace/jaeger"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/splunk"
)

func TestCreateTraceExporter(t *testing.T) {
	config := &Config{
		ExporterSettings:   configmodels.ExporterSettings{TypeVal: configmodels.Type(typeStr), NameVal: "sapm/customname"},
		Endpoint:           "test-endpoint",
		AccessToken:        "abcd1234",
		NumWorkers:         3,
		MaxConnections:     45,
		DisableCompression: true,
		AccessTokenPassthroughConfig: splunk.AccessTokenPassthroughConfig{
			AccessTokenPassthrough: true,
		},
	}
	params := component.ExporterCreateParams{Logger: zap.NewNop()}

	te, err := newSAPMTraceExporter(config, params)
	assert.Nil(t, err)
	assert.NotNil(t, te, "failed to create trace exporter")

	assert.NoError(t, te.Shutdown(context.Background()), "trace exporter shutdown failed")
}

func TestCreateTraceExporterWithInvalidConfig(t *testing.T) {
	config := &Config{}
	params := component.ExporterCreateParams{Logger: zap.NewNop()}
	te, err := newSAPMTraceExporter(config, params)
	require.Error(t, err)
	assert.Nil(t, te)
}

func buildTestTraces(setTokenLabel bool) (traces pdata.Traces) {
	traces = pdata.NewTraces()
	rss := traces.ResourceSpans()
	rss.Resize(20)

	for i := 0; i < 20; i++ {
		span := rss.At(i)
		resource := span.Resource()
		if setTokenLabel && i%2 == 1 {
			tokenLabel := fmt.Sprintf("MyToken%d", i/5)
			resource.Attributes().InsertString("com.splunk.signalfx.access_token", tokenLabel)
		}

		span.InstrumentationLibrarySpans().Resize(1)
		span.InstrumentationLibrarySpans().At(0).Spans().Resize(1)
		name := fmt.Sprintf("Span%d", i)
		span.InstrumentationLibrarySpans().At(0).Spans().At(0).SetName(name)
		span.InstrumentationLibrarySpans().At(0).Spans().At(0).SetTraceID(pdata.NewTraceID([16]byte{1}))
		span.InstrumentationLibrarySpans().At(0).Spans().At(0).SetSpanID(pdata.NewSpanID([8]byte{1}))
	}

	return traces
}

func TestFilterToken(t *testing.T) {
	tests := []struct {
		name     string
		useToken bool
	}{
		{
			name:     "no token",
			useToken: false,
		},
		{
			name:     "some with token",
			useToken: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			traces := buildTestTraces(tt.useToken)
			batches, err := jaeger.InternalTracesToJaegerProto(traces)
			require.NoError(t, err)
			assert.Equal(t, tt.useToken, hasToken(batches))
			filterToken(batches)
			assert.False(t, hasToken(batches))
		})
	}
}

func hasToken(batches []*model.Batch) bool {
	for _, batch := range batches {
		proc := batch.Process
		if proc == nil {
			continue
		}
		for i := range proc.Tags {
			if proc.Tags[i].Key == splunk.SFxAccessTokenLabel {
				return true
			}
		}
	}
	return false
}

func buildTestTrace(setIds bool) pdata.Traces {
	trace := pdata.NewTraces()
	trace.ResourceSpans().Resize(2)
	for i := 0; i < 2; i++ {
		span := trace.ResourceSpans().At(i)
		resource := span.Resource()
		resource.Attributes().InsertString("com.splunk.signalfx.access_token", fmt.Sprintf("TraceAccessToken%v", i))
		span.InstrumentationLibrarySpans().Resize(1)
		span.InstrumentationLibrarySpans().At(0).Spans().Resize(1)
		span.InstrumentationLibrarySpans().At(0).Spans().At(0).SetName("MySpan")

		rand.Seed(time.Now().Unix())
		var traceIDBytes [16]byte
		var spanIDBytes [8]byte
		rand.Read(traceIDBytes[:])
		rand.Read(spanIDBytes[:])
		if setIds {
			span.InstrumentationLibrarySpans().At(0).Spans().At(0).SetTraceID(pdata.NewTraceID(traceIDBytes))
			span.InstrumentationLibrarySpans().At(0).Spans().At(0).SetSpanID(pdata.NewSpanID(spanIDBytes))
		}
	}
	return trace
}

func TestSAPMClientTokenUsageAndErrorMarshalling(t *testing.T) {
	tests := []struct {
		name                   string
		accessTokenPassthrough bool
		translateError         bool
		sendError              bool
	}{
		{
			name:                   "no error without passthrough",
			accessTokenPassthrough: false,
			translateError:         false,
			sendError:              false,
		},
		{
			name:                   "no error with passthrough",
			accessTokenPassthrough: true,
			translateError:         false,
			sendError:              false,
		},
		{
			name:                   "translate error",
			accessTokenPassthrough: true,
			translateError:         true,
			sendError:              false,
		},
		{
			name:                   "sendError",
			accessTokenPassthrough: true,
			translateError:         false,
			sendError:              true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracesReceived := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedToken := "ClientAccessToken"
				if tt.accessTokenPassthrough {
					expectedToken = "TraceAccessToken"
				}
				assert.Contains(t, r.Header.Get("x-sf-token"), expectedToken)
				status := 200
				if tt.sendError {
					status = 400
				}
				w.WriteHeader(status)
				tracesReceived = true
			}))
			defer func() {
				if !tt.translateError {
					assert.True(t, tracesReceived, "Test server never received traces.")
				} else {
					assert.False(t, tracesReceived, "Test server received traces when none expected.")
				}
			}()
			defer server.Close()

			config := &Config{
				Endpoint:    server.URL,
				AccessToken: "ClientAccessToken",
				AccessTokenPassthroughConfig: splunk.AccessTokenPassthroughConfig{
					AccessTokenPassthrough: tt.accessTokenPassthrough,
				},
			}
			params := component.ExporterCreateParams{Logger: zap.NewNop()}

			se, err := newSAPMExporter(config, params)
			assert.Nil(t, err)
			assert.NotNil(t, se, "failed to create trace exporter")

			trace := buildTestTrace(!tt.translateError)
			err = se.pushTraceData(context.Background(), trace)

			if tt.sendError || tt.translateError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
