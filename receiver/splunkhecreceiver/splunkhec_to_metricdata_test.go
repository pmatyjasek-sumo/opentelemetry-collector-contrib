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

package splunkhecreceiver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/splunk"
)

func Test_splunkV2ToMetricsData(t *testing.T) {
	// Timestamps for Splunk have a resolution to the millisecond, where the time is reported in seconds with a floating value to the millisecond.
	now := time.Now()
	msecInt64 := now.UnixNano() / 1e6
	sec := float64(msecInt64) / 1e3
	nanos := int64(sec * 1e9)

	buildDefaultSplunkDataPt := func() *splunk.Event {
		return &splunk.Event{
			Time:       &sec,
			Host:       "localhost",
			Source:     "source",
			SourceType: "sourcetype",
			Index:      "index",
			Event:      "metrics",
			Fields: map[string]interface{}{
				"metric_name:single": int64Ptr(13),
				"k0":                 "v0",
				"k1":                 "v1",
				"k2":                 "v2",
			},
		}
	}

	tests := []struct {
		name                  string
		splunkDataPoint       *splunk.Event
		wantMetricsData       pdata.Metrics
		wantDroppedTimeseries int
	}{
		{
			name:            "int_gauge",
			splunkDataPoint: buildDefaultSplunkDataPt(),
			wantMetricsData: buildDefaultMetricsData(nanos),
		},
		{
			name: "multiple",
			splunkDataPoint: func() *splunk.Event {
				pt := buildDefaultSplunkDataPt()
				pt.Fields["metric_name:yetanother"] = int64Ptr(14)
				pt.Fields["metric_name:yetanotherandanother"] = int64Ptr(15)
				return pt
			}(),
			wantMetricsData: func() pdata.Metrics {
				metrics := buildDefaultMetricsData(nanos)
				mts := metrics.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics()
				mts.Resize(3)

				metricPt := mts.At(1)
				metricPt.SetDataType(pdata.MetricDataTypeIntGauge)
				metricPt.SetName("yetanother")
				metricPt.IntGauge().DataPoints().Resize(1)
				intPt := metricPt.IntGauge().DataPoints().At(0)
				intPt.SetValue(14)
				intPt.SetTimestamp(pdata.Timestamp(nanos))
				intPt.LabelsMap().Insert("k0", "v0")
				intPt.LabelsMap().Insert("k1", "v1")
				intPt.LabelsMap().Insert("k2", "v2")

				metricPt2 := mts.At(2)
				metricPt2.SetDataType(pdata.MetricDataTypeIntGauge)
				metricPt2.SetName("yetanotherandanother")
				metricPt2.IntGauge().DataPoints().Resize(1)
				intPt2 := metricPt2.IntGauge().DataPoints().At(0)
				intPt2.SetValue(15)
				intPt2.SetTimestamp(pdata.Timestamp(nanos))
				intPt2.LabelsMap().Insert("k0", "v0")
				intPt2.LabelsMap().Insert("k1", "v1")
				intPt2.LabelsMap().Insert("k2", "v2")

				return metrics
			}(),
		},
		{
			name: "double_gauge",
			splunkDataPoint: func() *splunk.Event {
				pt := buildDefaultSplunkDataPt()
				pt.Fields["metric_name:single"] = float64Ptr(13.13)
				return pt
			}(),
			wantMetricsData: func() pdata.Metrics {
				md := buildDefaultMetricsData(nanos)
				mts := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics()
				mts.Resize(1)
				metricPt := mts.At(0)
				metricPt.SetDataType(pdata.MetricDataTypeDoubleGauge)
				metricPt.SetName("single")
				metricPt.DoubleGauge().DataPoints().Resize(1)
				doublePt := metricPt.DoubleGauge().DataPoints().At(0)
				doublePt.SetValue(13.13)
				doublePt.SetTimestamp(pdata.Timestamp(nanos))
				doublePt.LabelsMap().Insert("k0", "v0")
				doublePt.LabelsMap().Insert("k1", "v1")
				doublePt.LabelsMap().Insert("k2", "v2")
				return md
			}(),
		},
		{
			name: "int_counter_pointer",
			splunkDataPoint: func() *splunk.Event {
				pt := buildDefaultSplunkDataPt()
				return pt
			}(),
			wantMetricsData: buildDefaultMetricsData(nanos),
		},
		{
			name: "int_counter",
			splunkDataPoint: func() *splunk.Event {
				pt := buildDefaultSplunkDataPt()
				pt.Fields["metric_name:single"] = int64(13)
				return pt
			}(),
			wantMetricsData: buildDefaultMetricsData(nanos),
		},
		{
			name: "double_counter",
			splunkDataPoint: func() *splunk.Event {
				pt := buildDefaultSplunkDataPt()
				pt.Fields["metric_name:single"] = float64Ptr(13.13)
				return pt
			}(),
			wantMetricsData: func() pdata.Metrics {
				md := buildDefaultMetricsData(nanos)
				mts := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics()
				mts.Resize(1)
				metricPt := mts.At(0)
				metricPt.SetDataType(pdata.MetricDataTypeDoubleGauge)
				metricPt.SetName("single")
				metricPt.DoubleGauge().DataPoints().Resize(1)
				doublePt := metricPt.DoubleGauge().DataPoints().At(0)
				doublePt.SetValue(13.13)
				doublePt.LabelsMap().Insert("k0", "v0")
				doublePt.LabelsMap().Insert("k1", "v1")
				doublePt.LabelsMap().Insert("k2", "v2")
				doublePt.SetTimestamp(pdata.Timestamp(nanos))
				return md
			}(),
		},
		{
			name: "double_counter_as_string_pointer",
			splunkDataPoint: func() *splunk.Event {
				pt := buildDefaultSplunkDataPt()
				pt.Fields["metric_name:single"] = strPtr("13.13")
				return pt
			}(),
			wantMetricsData: func() pdata.Metrics {
				md := buildDefaultMetricsData(nanos)
				mts := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics()
				mts.Resize(1)
				metricPt := mts.At(0)
				metricPt.SetDataType(pdata.MetricDataTypeDoubleGauge)
				metricPt.SetName("single")
				metricPt.DoubleGauge().DataPoints().Resize(1)
				doublePt := metricPt.DoubleGauge().DataPoints().At(0)
				doublePt.SetValue(13.13)
				doublePt.LabelsMap().Insert("k0", "v0")
				doublePt.LabelsMap().Insert("k1", "v1")
				doublePt.LabelsMap().Insert("k2", "v2")
				doublePt.SetTimestamp(pdata.Timestamp(nanos))
				return md
			}(),
		},
		{
			name: "double_counter_as_string",
			splunkDataPoint: func() *splunk.Event {
				pt := buildDefaultSplunkDataPt()
				pt.Fields["metric_name:single"] = "13.13"
				return pt
			}(),
			wantMetricsData: func() pdata.Metrics {
				md := buildDefaultMetricsData(nanos)
				mts := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics()
				mts.Resize(1)
				metricPt := mts.At(0)
				metricPt.SetDataType(pdata.MetricDataTypeDoubleGauge)
				metricPt.SetName("single")
				metricPt.DoubleGauge().DataPoints().Resize(1)
				doublePt := metricPt.DoubleGauge().DataPoints().At(0)
				doublePt.SetValue(13.13)
				doublePt.LabelsMap().Insert("k0", "v0")
				doublePt.LabelsMap().Insert("k1", "v1")
				doublePt.LabelsMap().Insert("k2", "v2")
				doublePt.SetTimestamp(pdata.Timestamp(nanos))
				return md
			}(),
		},
		{
			name: "zero_timestamp",
			splunkDataPoint: func() *splunk.Event {
				pt := buildDefaultSplunkDataPt()
				pt.Time = new(float64)
				return pt
			}(),
			wantMetricsData: func() pdata.Metrics {
				return buildDefaultMetricsData(0)
			}(),
		},
		{
			name: "empty_dimension_value",
			splunkDataPoint: func() *splunk.Event {
				pt := buildDefaultSplunkDataPt()
				pt.Fields["k0"] = ""
				return pt
			}(),
			wantMetricsData: func() pdata.Metrics {
				md := buildDefaultMetricsData(nanos)
				md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).IntGauge().DataPoints().At(0).LabelsMap().Update("k0", "")
				return md
			}(),
		},
		{
			name: "invalid_point",
			splunkDataPoint: func() *splunk.Event {
				pt := buildDefaultSplunkDataPt()
				pt.Fields["metric_name:single"] = "foo"
				return pt
			}(),
			wantMetricsData: func() pdata.Metrics {
				return pdata.NewMetrics()
			}(),
			wantDroppedTimeseries: 1,
		},
		{
			name: "cannot_convert_string",
			splunkDataPoint: func() *splunk.Event {
				pt := buildDefaultSplunkDataPt()
				value := "foo"
				pt.Fields["metric_name:single"] = &value
				return pt
			}(),
			wantMetricsData: func() pdata.Metrics {
				return pdata.NewMetrics()
			}(),
			wantDroppedTimeseries: 1,
		},
		{
			name: "nil_dimension_ignored",
			splunkDataPoint: func() *splunk.Event {
				pt := buildDefaultSplunkDataPt()
				pt.Fields["k4"] = nil
				pt.Fields["k5"] = nil
				pt.Fields["k6"] = nil
				return pt
			}(),
			wantMetricsData: buildDefaultMetricsData(nanos),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md, numDroppedTimeseries := SplunkHecToMetricsData(zap.NewNop(), []*splunk.Event{tt.splunkDataPoint}, func(resource pdata.Resource) {})
			assert.Equal(t, tt.wantDroppedTimeseries, numDroppedTimeseries)
			assert.Equal(t, tt.wantMetricsData, md)
		})
	}
}

func buildDefaultMetricsData(time int64) pdata.Metrics {
	metrics := pdata.NewMetrics()
	metrics.ResourceMetrics().Resize(1)
	resourceMetrics := metrics.ResourceMetrics().At(0)
	attrs := resourceMetrics.Resource().Attributes()
	attrs.InsertString("host.name", "localhost")
	attrs.InsertString("service.name", "source")
	attrs.InsertString("com.splunk.sourcetype", "sourcetype")
	attrs.InsertString("com.splunk.index", "index")

	resourceMetrics.InstrumentationLibraryMetrics().Resize(1)
	ilm := resourceMetrics.InstrumentationLibraryMetrics().At(0)
	ilm.Metrics().Resize(1)
	metricPt := ilm.Metrics().At(0)
	metricPt.SetDataType(pdata.MetricDataTypeIntGauge)
	metricPt.SetName("single")
	metricPt.IntGauge().DataPoints().Resize(1)
	intPt := metricPt.IntGauge().DataPoints().At(0)
	intPt.SetValue(13)
	intPt.LabelsMap().Insert("k0", "v0")
	intPt.LabelsMap().Insert("k1", "v1")
	intPt.LabelsMap().Insert("k2", "v2")
	intPt.SetTimestamp(pdata.Timestamp(time))
	return metrics
}

func strPtr(s string) *string {
	l := s
	return &l
}

func int64Ptr(i int64) *int64 {
	l := i
	return &l
}

func float64Ptr(f float64) *float64 {
	l := f
	return &l
}
