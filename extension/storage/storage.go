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

package storage

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
)

// Extension is the interface that storage extensions must implement
type Extension interface {
	component.Extension

	// GetClient will create a client for use by the specified component.
	// The component can use the client to manage state
	// TODO change parameters to new config.ComponentID
	// https://github.com/open-telemetry/opentelemetry-collector/pull/2869
	GetClient(context.Context, component.Kind, configmodels.NamedEntity) (Client, error)
}

// Client is the interface that storage clients must implement
// All methods should return error only if a problem occurred
type Client interface {

	// Get will retrieve data from storage that corresponds to the
	// specified key. It should return nil, nil if not found
	Get(context.Context, string) ([]byte, error)

	// Set will store data. The data can be retrieved by the same
	// component after a process restart, using the same key
	Set(context.Context, string, []byte) error

	// Delete will delete data associated with the specified key
	Delete(context.Context, string) error
}
