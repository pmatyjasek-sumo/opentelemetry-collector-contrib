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

package carbonexporter

import (
	"time"

	"go.opentelemetry.io/collector/config"
)

// Defaults for not specified configuration settings.
const (
	DefaultEndpoint    = "localhost:2003"
	DefaultSendTimeout = 5 * time.Second
)

// Config defines configuration for Carbon exporter.
type Config struct {
	config.ExporterSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.

	// Endpoint specifies host and port to send metrics in the Carbon plaintext
	// format. The default value is defined by the DefaultEndpoint constant.
	Endpoint string `mapstructure:"endpoint"`

	// Timeout is the maximum duration allowed to connecting and sending the
	// data to the Carbon/Graphite backend.
	// The default value is defined by the DefaultSendTimeout constant.
	Timeout time.Duration `mapstructure:"timeout"`
}
