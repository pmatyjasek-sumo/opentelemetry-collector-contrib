// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dotnetdiagnosticsreceiver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configcheck"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.uber.org/zap"
)

func TestCreateDefaultConfig(t *testing.T) {
	f := NewFactory()
	assert.Equal(t, "dotnet_diagnostics", string(f.Type()))
	cfg := f.CreateDefaultConfig()
	assert.NotNil(t, cfg)
	assert.NoError(t, configcheck.ValidateConfig(cfg))
	assert.Equal(
		t,
		[]string{"System.Runtime", "Microsoft.AspNetCore.Hosting"},
		cfg.(*Config).Counters,
	)
}

func TestCreateReceiver(t *testing.T) {
	f := NewFactory()
	cfg := f.CreateDefaultConfig()
	params := component.ReceiverCreateParams{Logger: zap.NewNop()}
	r, err := f.CreateMetricsReceiver(context.Background(), params, cfg, consumertest.NewMetricsNop())
	assert.NoError(t, err)
	assert.NotNil(t, r)
}
