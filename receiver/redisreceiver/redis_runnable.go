// Copyright 2020, OpenTelemetry Authors
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

package redisreceiver

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/obsreport"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/redisreceiver/interval"
)

var _ interval.Runnable = (*redisRunnable)(nil)

// Runs intermittently, fetching info from Redis, creating metrics/datapoints,
// and feeding them to a metricsConsumer.
type redisRunnable struct {
	ctx             context.Context
	metricsConsumer consumer.Metrics
	redisSvc        *redisSvc
	redisMetrics    []*redisMetric
	logger          *zap.Logger
	timeBundle      *timeBundle
	serviceName     string
}

func newRedisRunnable(
	ctx context.Context,
	client client,
	serviceName string,
	metricsConsumer consumer.Metrics,
	logger *zap.Logger,
) *redisRunnable {
	return &redisRunnable{
		ctx:             ctx,
		serviceName:     serviceName,
		redisSvc:        newRedisSvc(client),
		metricsConsumer: metricsConsumer,
		logger:          logger,
	}
}

// Builds a data structure of all of the keys, types, converters and such to
// later extract data from Redis.
func (r *redisRunnable) Setup() error {
	r.redisMetrics = getDefaultRedisMetrics()
	return nil
}

// Run is called periodically, querying Redis and building Metrics to send to
// the next consumer. First builds 'fixed' metrics (non-keyspace metrics)
// defined at startup time. Then builds 'keyspace' metrics if there are any
// keyspace lines returned by Redis. There should be one keyspace line per
// active Redis database, of which there can be 16.
func (r *redisRunnable) Run() error {
	const dataFormat = "redis"
	const transport = "http" // todo verify this
	ctx := obsreport.StartMetricsReceiveOp(r.ctx, dataFormat, transport)

	inf, err := r.redisSvc.info()
	if err != nil {
		obsreport.EndMetricsReceiveOp(ctx, dataFormat, 0, err)
		return nil
	}

	uptime, err := inf.getUptimeInSeconds()
	if err != nil {
		obsreport.EndMetricsReceiveOp(ctx, dataFormat, 0, err)
		return nil
	}

	if r.timeBundle == nil {
		r.timeBundle = newTimeBundle(time.Now(), uptime)
	} else {
		r.timeBundle.update(time.Now(), uptime)
	}

	pdm := pdata.NewMetrics()
	rms := pdm.ResourceMetrics()

	rm := pdata.NewResourceMetrics()
	rms.Append(rm)
	resource := rm.Resource()
	rattrs := resource.Attributes()
	rattrs.InsertString("service.name", r.serviceName)

	ilms := rm.InstrumentationLibraryMetrics()

	ilm := pdata.NewInstrumentationLibraryMetrics()
	ilms.Append(ilm)

	fixedMS, warnings := inf.buildFixedMetrics(r.redisMetrics, r.timeBundle)
	fixedMS.MoveAndAppendTo(ilm.Metrics())
	if warnings != nil {
		r.logger.Warn(
			"errors parsing redis string",
			zap.Errors("parsing errors", warnings),
		)
	}

	keyspaceMS, warnings := inf.buildKeyspaceMetrics(r.timeBundle)
	if warnings != nil {
		r.logger.Warn(
			"errors parsing keyspace string",
			zap.Errors("parsing errors", warnings),
		)
	}
	keyspaceMS.MoveAndAppendTo(ilm.Metrics())

	err = r.metricsConsumer.ConsumeMetrics(r.ctx, pdm)
	_, numPoints := pdm.MetricAndDataPointCount()
	obsreport.EndMetricsReceiveOp(ctx, dataFormat, numPoints, err)

	return nil
}
