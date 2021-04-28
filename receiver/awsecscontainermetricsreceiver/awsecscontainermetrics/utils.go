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

package awsecscontainermetrics

import (
	"strconv"
	"time"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"go.opentelemetry.io/collector/testutil/metricstestutil"
	"go.opentelemetry.io/collector/translator/internaldata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GenerateDummyMetrics generates two dummy metrics
// TODO: Remove once this is replaced with actuall metrics generation code
func GenerateDummyMetrics() internaldata.MetricsData {
	md := internaldata.MetricsData{}

	for i := 0; i < 2; i++ {
		md.Metrics = append(md.Metrics, createGaugeIntMetric(i))
	}
	return md
}

// createGaugeIntMetric creates a int gauge metric
func createGaugeIntMetric(i int) *metricspb.Metric {
	ts := time.Now()
	return metricstestutil.GaugeInt(
		"test_metric_"+strconv.Itoa(i),
		[]string{"label_key_0", "label_key_1"},
		metricstestutil.Timeseries(
			time.Now(),
			[]string{"label_value_0", "label_value_1"},
			&metricspb.Point{
				Timestamp: timestamppb.New(ts),
				Value:     &metricspb.Point_Int64Value{Int64Value: int64(i)},
			},
		),
	)
}

func timestampProto(t time.Time) *timestamp.Timestamp {
	out, _ := ptypes.TimestampProto(t)
	return out
}

func applyTimestamp(metrics []*metricspb.Metric, t *timestamp.Timestamp) []*metricspb.Metric {
	for _, metric := range metrics {
		if metric != nil {
			metric.Timeseries[0].Points[0].Timestamp = t
		}
	}
	return metrics
}
