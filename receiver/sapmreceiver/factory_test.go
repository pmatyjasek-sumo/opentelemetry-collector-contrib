// Copyright 2019, OpenTelemetry Authors
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

package sapmreceiver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/config/configcheck"
	"go.uber.org/zap"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	assert.NoError(t, configcheck.ValidateConfig(cfg))
}

func TestCreateReceiver(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	componentSettings := component.ComponentSettings{Logger: zap.NewNop()}
	tReceiver, err := factory.CreateTracesReceiver(context.Background(), componentSettings, cfg, nil)
	assert.NoError(t, err, "receiver creation failed")
	assert.NotNil(t, tReceiver, "receiver creation failed")

	mReceiver, err := factory.CreateMetricsReceiver(context.Background(), componentSettings, cfg, nil)
	assert.ErrorIs(t, err, componenterror.ErrDataTypeIsNotSupported)
	assert.Nil(t, mReceiver)
}

func TestCreateInvalidHTTPEndpoint(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	rCfg := cfg.(*Config)

	rCfg.Endpoint = ""
	componentSettings := component.ComponentSettings{Logger: zap.NewNop()}
	_, err := factory.CreateTracesReceiver(context.Background(), componentSettings, cfg, nil)
	assert.Error(t, err, "receiver creation with no endpoints must fail")
}

func TestCreateNoPort(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	rCfg := cfg.(*Config)

	rCfg.Endpoint = "localhost:"
	componentSettings := component.ComponentSettings{Logger: zap.NewNop()}
	_, err := factory.CreateTracesReceiver(context.Background(), componentSettings, cfg, nil)
	assert.Error(t, err, "receiver creation with no port number must fail")
}

func TestCreateLargePort(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	rCfg := cfg.(*Config)

	rCfg.Endpoint = "localhost:65536"
	componentSettings := component.ComponentSettings{Logger: zap.NewNop()}
	_, err := factory.CreateTracesReceiver(context.Background(), componentSettings, cfg, nil)
	assert.Error(t, err, "receiver creation with too large port number must fail")
}
