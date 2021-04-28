// Copyright 2020, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package awsemfexporter

import (
	"reflect"
	"strings"
	"testing"
	"time"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/translator/internaldata"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/aws"
)

func generateTestIntGauge(name string) *metricspb.Metric {
	return &metricspb.Metric{
		MetricDescriptor: &metricspb.MetricDescriptor{
			Name: name,
			Type: metricspb.MetricDescriptor_GAUGE_INT64,
			Unit: "Count",
			LabelKeys: []*metricspb.LabelKey{
				{Key: "label1"},
			},
		},
		Timeseries: []*metricspb.TimeSeries{
			{
				LabelValues: []*metricspb.LabelValue{
					{Value: "value1", HasValue: true},
				},
				Points: []*metricspb.Point{
					{
						Value: &metricspb.Point_Int64Value{
							Int64Value: 1,
						},
					},
				},
			},
		},
	}
}

func generateTestDoubleGauge(name string) *metricspb.Metric {
	return &metricspb.Metric{
		MetricDescriptor: &metricspb.MetricDescriptor{
			Name: name,
			Type: metricspb.MetricDescriptor_GAUGE_DOUBLE,
			Unit: "Count",
			LabelKeys: []*metricspb.LabelKey{
				{Key: "label1"},
			},
		},
		Timeseries: []*metricspb.TimeSeries{
			{
				LabelValues: []*metricspb.LabelValue{
					{Value: "value1", HasValue: true},
				},
				Points: []*metricspb.Point{
					{
						Value: &metricspb.Point_DoubleValue{
							DoubleValue: 0.1,
						},
					},
				},
			},
		},
	}
}

func generateTestIntSum(name string) *metricspb.Metric {
	return &metricspb.Metric{
		MetricDescriptor: &metricspb.MetricDescriptor{
			Name: name,
			Type: metricspb.MetricDescriptor_CUMULATIVE_INT64,
			Unit: "Count",
			LabelKeys: []*metricspb.LabelKey{
				{Key: "label1"},
			},
		},
		Timeseries: []*metricspb.TimeSeries{
			{
				LabelValues: []*metricspb.LabelValue{
					{Value: "value1", HasValue: true},
					{Value: "value2", HasValue: true},
				},
				Points: []*metricspb.Point{
					{
						Value: &metricspb.Point_Int64Value{
							Int64Value: 1,
						},
					},
				},
			},
		},
	}
}

func generateTestDoubleSum(name string) *metricspb.Metric {
	return &metricspb.Metric{
		MetricDescriptor: &metricspb.MetricDescriptor{
			Name: name,
			Type: metricspb.MetricDescriptor_CUMULATIVE_DOUBLE,
			Unit: "Count",
			LabelKeys: []*metricspb.LabelKey{
				{Key: "label1"},
			},
		},
		Timeseries: []*metricspb.TimeSeries{
			{
				LabelValues: []*metricspb.LabelValue{
					{Value: "value1", HasValue: true},
					{Value: "value2", HasValue: true},
				},
				Points: []*metricspb.Point{
					{
						Value: &metricspb.Point_DoubleValue{
							DoubleValue: 0.1,
						},
					},
				},
			},
		},
	}
}

func generateTestHistogram(name string) *metricspb.Metric {
	return &metricspb.Metric{
		MetricDescriptor: &metricspb.MetricDescriptor{
			Name: name,
			Type: metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION,
			Unit: "Seconds",
			LabelKeys: []*metricspb.LabelKey{
				{Key: "label1"},
			},
		},
		Timeseries: []*metricspb.TimeSeries{
			{
				LabelValues: []*metricspb.LabelValue{
					{Value: "value1", HasValue: true},
					{Value: "value2", HasValue: true},
				},
				Points: []*metricspb.Point{
					{
						Value: &metricspb.Point_DistributionValue{
							DistributionValue: &metricspb.DistributionValue{
								Sum:   35.0,
								Count: 18,
								BucketOptions: &metricspb.DistributionValue_BucketOptions{
									Type: &metricspb.DistributionValue_BucketOptions_Explicit_{
										Explicit: &metricspb.DistributionValue_BucketOptions_Explicit{
											Bounds: []float64{0, 10},
										},
									},
								},
								Buckets: []*metricspb.DistributionValue_Bucket{
									{
										Count: 5,
									},
									{
										Count: 6,
									},
									{
										Count: 7,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func generateTestSummary(name string) *metricspb.Metric {
	return &metricspb.Metric{
		MetricDescriptor: &metricspb.MetricDescriptor{
			Name: name,
			Type: metricspb.MetricDescriptor_SUMMARY,
			Unit: "Seconds",
			LabelKeys: []*metricspb.LabelKey{
				{Key: "label1"},
			},
		},
		Timeseries: []*metricspb.TimeSeries{
			{
				LabelValues: []*metricspb.LabelValue{
					{Value: "value1", HasValue: true},
				},
				Points: []*metricspb.Point{
					{
						Value: &metricspb.Point_SummaryValue{
							SummaryValue: &metricspb.SummaryValue{
								Sum: &wrappers.DoubleValue{
									Value: 15.0,
								},
								Count: &wrappers.Int64Value{
									Value: 5,
								},
								Snapshot: &metricspb.SummaryValue_Snapshot{
									Count: &wrappers.Int64Value{
										Value: 5,
									},
									Sum: &wrappers.DoubleValue{
										Value: 15.0,
									},
									PercentileValues: []*metricspb.SummaryValue_Snapshot_ValueAtPercentile{
										{
											Percentile: 0.0,
											Value:      1,
										},
										{
											Percentile: 100.0,
											Value:      5,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func setupDataPointCache() {
	deltaMetricCalculator = aws.NewFloat64DeltaCalculator()
	summaryMetricCalculator = aws.NewMetricCalculator(calculateSummaryDelta)
}

func TestIntDataPointSliceAt(t *testing.T) {
	setupDataPointCache()

	instrLibName := "cloudwatch-otel"
	labels := map[string]string{"label": "value"}

	testDeltaCases := []struct {
		testName        string
		adjustToDelta   bool
		value           interface{}
		calculatedValue interface{}
	}{
		{
			"w/ 1st delta calculation",
			true,
			int64(-17),
			float64(-17),
		},
		{
			"w/ 2st delta calculation",
			true,
			int64(1),
			float64(18),
		},
	}

	for _, tc := range testDeltaCases {
		t.Run(tc.testName, func(t *testing.T) {
			testDPS := pdata.NewIntDataPointSlice()
			testDPS.Resize(1)
			testDP := testDPS.At(0)
			testDP.SetValue(tc.value.(int64))
			testDP.LabelsMap().InitFromMap(labels)

			dps := IntDataPointSlice{
				instrLibName,
				deltaMetricMetadata{
					tc.adjustToDelta,
					"foo",
					0,
					"namespace",
					"log-group",
					"log-stream",
				},
				testDPS,
			}

			expectedDP := DataPoint{
				Value: tc.calculatedValue,
				Labels: map[string]string{
					oTellibDimensionKey: instrLibName,
					"label":             "value",
				},
			}

			assert.Equal(t, 1, dps.Len())
			dp := dps.At(0)
			if strings.Contains(tc.testName, "2nd rate") {
				assert.InDelta(t, expectedDP.Value.(float64), dp.Value.(float64), 0.02)
			} else {
				assert.Equal(t, expectedDP, dp)
			}
		})
	}
}

func TestDoubleDataPointSliceAt(t *testing.T) {
	setupDataPointCache()

	instrLibName := "cloudwatch-otel"
	labels := map[string]string{"label1": "value1"}

	testDeltaCases := []struct {
		testName        string
		adjustToDelta   bool
		value           interface{}
		calculatedValue interface{}
	}{
		{
			"w/ 1st delta calculation",
			true,
			float64(0.4),
			float64(0.4),
		},
		{
			"w/ 2nd delta calculation",
			false,
			float64(0.5),
			float64(0.1),
		},
	}

	for _, tc := range testDeltaCases {
		t.Run(tc.testName, func(t *testing.T) {
			testDPS := pdata.NewDoubleDataPointSlice()
			testDPS.Resize(1)
			testDP := testDPS.At(0)
			testDP.SetValue(tc.value.(float64))
			testDP.LabelsMap().InitFromMap(labels)

			dps := DoubleDataPointSlice{
				instrLibName,
				deltaMetricMetadata{
					tc.adjustToDelta,
					"foo",
					0,
					"namespace",
					"log-group",
					"log-stream",
				},
				testDPS,
			}

			expectedDP := DataPoint{
				Value: tc.calculatedValue,
				Labels: map[string]string{
					oTellibDimensionKey: instrLibName,
					"label1":            "value1",
				},
			}

			assert.Equal(t, 1, dps.Len())
			dp := dps.At(0)
			assert.True(t, (expectedDP.Value.(float64)-dp.Value.(float64)) < 0.002)
		})
	}
}

func TestHistogramDataPointSliceAt(t *testing.T) {
	instrLibName := "cloudwatch-otel"
	labels := map[string]string{"label1": "value1"}

	testDPS := pdata.NewHistogramDataPointSlice()
	testDPS.Resize(1)
	testDP := testDPS.At(0)
	testDP.SetCount(uint64(17))
	testDP.SetSum(float64(17.13))
	testDP.SetBucketCounts([]uint64{1, 2, 3})
	testDP.SetExplicitBounds([]float64{1, 2, 3})
	testDP.LabelsMap().InitFromMap(labels)

	dps := HistogramDataPointSlice{
		instrLibName,
		testDPS,
	}

	expectedDP := DataPoint{
		Value: &CWMetricStats{
			Sum:   17.13,
			Count: 17,
		},
		Labels: map[string]string{
			oTellibDimensionKey: instrLibName,
			"label1":            "value1",
		},
	}

	assert.Equal(t, 1, dps.Len())
	dp := dps.At(0)
	assert.Equal(t, expectedDP, dp)
}

func TestSummaryDataPointSliceAt(t *testing.T) {
	setupDataPointCache()

	instrLibName := "cloudwatch-otel"
	labels := map[string]string{"label1": "value1"}
	metadataTimeStamp := time.Now().UnixNano() / int64(time.Millisecond)

	testCases := []struct {
		testName           string
		inputSumCount      []interface{}
		calculatedSumCount []interface{}
	}{
		{
			"1st summary count calculation",
			[]interface{}{float64(17.3), uint64(17)},
			[]interface{}{float64(17.3), uint64(17)},
		},
		{
			"2nd summary count calculation",
			[]interface{}{float64(100), uint64(25)},
			[]interface{}{float64(82.7), uint64(8)},
		},
		{
			"3rd summary count calculation",
			[]interface{}{float64(120), uint64(26)},
			[]interface{}{float64(20), uint64(1)},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.testName, func(t *testing.T) {
			testDPS := pdata.NewSummaryDataPointSlice()
			testDPS.Resize(1)
			testDP := testDPS.At(0)
			testDP.SetSum(tt.inputSumCount[0].(float64))
			testDP.SetCount(tt.inputSumCount[1].(uint64))

			testDP.QuantileValues().Resize(2)
			testQuantileValue := testDP.QuantileValues().At(0)
			testQuantileValue.SetQuantile(0)
			testQuantileValue.SetValue(float64(1))
			testQuantileValue = testDP.QuantileValues().At(1)
			testQuantileValue.SetQuantile(100)
			testQuantileValue.SetValue(float64(5))
			testDP.LabelsMap().InitFromMap(labels)

			dps := SummaryDataPointSlice{
				instrLibName,
				deltaMetricMetadata{
					true,
					"foo",
					metadataTimeStamp,
					"namespace",
					"log-group",
					"log-stream",
				},
				testDPS,
			}

			expectedDP := DataPoint{
				Value: &CWMetricStats{
					Max:   5,
					Min:   1,
					Sum:   tt.calculatedSumCount[0].(float64),
					Count: tt.calculatedSumCount[1].(uint64),
				},
				Labels: map[string]string{
					oTellibDimensionKey: instrLibName,
					"label1":            "value1",
				},
			}

			assert.Equal(t, 1, dps.Len())
			dp := dps.At(0)
			expectedMetricStats := expectedDP.Value.(*CWMetricStats)
			actualMetricsStats := dp.Value.(*CWMetricStats)
			assert.Equal(t, expectedDP.Labels, dp.Labels)
			assert.Equal(t, expectedMetricStats.Max, actualMetricsStats.Max)
			assert.Equal(t, expectedMetricStats.Min, actualMetricsStats.Min)
			assert.InDelta(t, expectedMetricStats.Count, actualMetricsStats.Count, 0.1)
			assert.True(t, expectedMetricStats.Sum-actualMetricsStats.Sum < float64(0.02))
		})
	}
}

func TestCreateLabels(t *testing.T) {
	expectedLabels := map[string]string{
		"a": "A",
		"b": "B",
		"c": "C",
	}
	labelsMap := pdata.NewStringMap().InitFromMap(expectedLabels)

	labels := createLabels(labelsMap, noInstrumentationLibraryName)
	assert.Equal(t, expectedLabels, labels)

	// With isntrumentation library name
	labels = createLabels(labelsMap, "cloudwatch-otel")
	expectedLabels[oTellibDimensionKey] = "cloudwatch-otel"
	assert.Equal(t, expectedLabels, labels)
}

func TestGetDataPoints(t *testing.T) {
	metadata := CWMetricMetadata{
		GroupedMetricMetadata: GroupedMetricMetadata{
			Namespace:   "namespace",
			TimestampMs: time.Now().UnixNano() / int64(time.Millisecond),
			LogGroup:    "log-group",
			LogStream:   "log-stream",
		},
		InstrumentationLibraryName: "cloudwatch-otel",
	}

	dmm := deltaMetricMetadata{
		false,
		"foo",
		metadata.TimestampMs,
		"namespace",
		"log-group",
		"log-stream",
	}
	cumulativeDmm := deltaMetricMetadata{
		true,
		"foo",
		metadata.TimestampMs,
		"namespace",
		"log-group",
		"log-stream",
	}
	testCases := []struct {
		testName           string
		metric             *metricspb.Metric
		expectedDataPoints DataPoints
	}{
		{
			"Int gauge",
			generateTestIntGauge("foo"),
			IntDataPointSlice{
				metadata.InstrumentationLibraryName,
				dmm,
				pdata.IntDataPointSlice{},
			},
		},
		{
			"Double gauge",
			generateTestDoubleGauge("foo"),
			DoubleDataPointSlice{
				metadata.InstrumentationLibraryName,
				dmm,
				pdata.DoubleDataPointSlice{},
			},
		},
		{
			"Int sum",
			generateTestIntSum("foo"),
			IntDataPointSlice{
				metadata.InstrumentationLibraryName,
				cumulativeDmm,
				pdata.IntDataPointSlice{},
			},
		},
		{
			"Double sum",
			generateTestDoubleSum("foo"),
			DoubleDataPointSlice{
				metadata.InstrumentationLibraryName,
				cumulativeDmm,
				pdata.DoubleDataPointSlice{},
			},
		},
		{
			"Double histogram",
			generateTestHistogram("foo"),
			HistogramDataPointSlice{
				metadata.InstrumentationLibraryName,
				pdata.HistogramDataPointSlice{},
			},
		},
		{
			"Summary",
			generateTestSummary("foo"),
			SummaryDataPointSlice{
				metadata.InstrumentationLibraryName,
				cumulativeDmm,
				pdata.SummaryDataPointSlice{},
			},
		},
	}

	for _, tc := range testCases {
		oc := internaldata.MetricsData{
			Metrics: []*metricspb.Metric{tc.metric},
		}

		// Retrieve *pdata.Metric
		rm := internaldata.OCToMetrics(oc).ResourceMetrics().At(0)
		metric := rm.InstrumentationLibraryMetrics().At(0).Metrics().At(0)

		logger := zap.NewNop()

		expectedLabels := pdata.NewStringMap().InitFromMap(map[string]string{"label1": "value1"})

		t.Run(tc.testName, func(t *testing.T) {
			dps := getDataPoints(&metric, metadata, logger)
			assert.NotNil(t, dps)
			assert.Equal(t, reflect.TypeOf(tc.expectedDataPoints), reflect.TypeOf(dps))
			switch convertedDPS := dps.(type) {
			case IntDataPointSlice:
				expectedDPS := tc.expectedDataPoints.(IntDataPointSlice)
				assert.Equal(t, metadata.InstrumentationLibraryName, convertedDPS.instrumentationLibraryName)
				assert.Equal(t, expectedDPS.deltaMetricMetadata, convertedDPS.deltaMetricMetadata)
				assert.Equal(t, 1, convertedDPS.Len())
				dp := convertedDPS.IntDataPointSlice.At(0)
				assert.Equal(t, int64(1), dp.Value())
				assert.Equal(t, expectedLabels, dp.LabelsMap())
			case DoubleDataPointSlice:
				expectedDPS := tc.expectedDataPoints.(DoubleDataPointSlice)
				assert.Equal(t, metadata.InstrumentationLibraryName, convertedDPS.instrumentationLibraryName)
				assert.Equal(t, expectedDPS.deltaMetricMetadata, convertedDPS.deltaMetricMetadata)
				assert.Equal(t, 1, convertedDPS.Len())
				dp := convertedDPS.DoubleDataPointSlice.At(0)
				assert.Equal(t, 0.1, dp.Value())
				assert.Equal(t, expectedLabels, dp.LabelsMap())
			case HistogramDataPointSlice:
				assert.Equal(t, metadata.InstrumentationLibraryName, convertedDPS.instrumentationLibraryName)
				assert.Equal(t, 1, convertedDPS.Len())
				dp := convertedDPS.HistogramDataPointSlice.At(0)
				assert.Equal(t, 35.0, dp.Sum())
				assert.Equal(t, uint64(18), dp.Count())
				assert.Equal(t, []float64{0, 10}, dp.ExplicitBounds())
				assert.Equal(t, expectedLabels, dp.LabelsMap())
			case SummaryDataPointSlice:
				assert.Equal(t, metadata.InstrumentationLibraryName, convertedDPS.instrumentationLibraryName)
				assert.Equal(t, 1, convertedDPS.Len())
				dp := convertedDPS.SummaryDataPointSlice.At(0)
				assert.Equal(t, 15.0, dp.Sum())
				assert.Equal(t, uint64(5), dp.Count())
				assert.Equal(t, 2, dp.QuantileValues().Len())
				assert.Equal(t, float64(1), dp.QuantileValues().At(0).Value())
				assert.Equal(t, float64(5), dp.QuantileValues().At(1).Value())
			}
		})
	}

	t.Run("Unhandled metric type", func(t *testing.T) {
		metric := pdata.NewMetric()
		metric.SetName("foo")
		metric.SetUnit("Count")
		metric.SetDataType(pdata.MetricDataTypeIntHistogram)

		obs, logs := observer.New(zap.WarnLevel)
		logger := zap.New(obs)

		dps := getDataPoints(&metric, metadata, logger)
		assert.Nil(t, dps)

		// Test output warning logs
		expectedLogs := []observer.LoggedEntry{
			{
				Entry: zapcore.Entry{Level: zap.WarnLevel, Message: "Unhandled metric data type."},
				Context: []zapcore.Field{
					zap.String("DataType", "IntHistogram"),
					zap.String("Name", "foo"),
					zap.String("Unit", "Count"),
				},
			},
		}
		assert.Equal(t, 1, logs.Len())
		assert.Equal(t, expectedLogs, logs.AllUntimed())
	})

	t.Run("Nil metric", func(t *testing.T) {
		dps := getDataPoints(nil, metadata, zap.NewNop())
		assert.Nil(t, dps)
	})
}

func BenchmarkGetDataPoints(b *testing.B) {
	oc := internaldata.MetricsData{
		Metrics: []*metricspb.Metric{
			generateTestIntGauge("int-gauge"),
			generateTestDoubleGauge("double-gauge"),
			generateTestIntSum("int-sum"),
			generateTestDoubleSum("double-sum"),
			generateTestHistogram("double-histogram"),
			generateTestSummary("summary"),
		},
	}
	rms := internaldata.OCToMetrics(oc).ResourceMetrics()
	metrics := rms.At(0).InstrumentationLibraryMetrics().At(0).Metrics()
	numMetrics := metrics.Len()

	metadata := CWMetricMetadata{
		GroupedMetricMetadata: GroupedMetricMetadata{
			Namespace:   "Namespace",
			TimestampMs: int64(1596151098037),
			LogGroup:    "log-group",
			LogStream:   "log-stream",
		},
		InstrumentationLibraryName: "cloudwatch-otel",
	}

	logger := zap.NewNop()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for i := 0; i < numMetrics; i++ {
			metric := metrics.At(i)
			getDataPoints(&metric, metadata, logger)
		}
	}
}

func TestIntDataPointSlice_At(t *testing.T) {
	type fields struct {
		instrumentationLibraryName string
		deltaMetricMetadata        deltaMetricMetadata
		IntDataPointSlice          pdata.IntDataPointSlice
	}
	type args struct {
		i int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   DataPoint
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dps := IntDataPointSlice{
				instrumentationLibraryName: tt.fields.instrumentationLibraryName,
				deltaMetricMetadata:        tt.fields.deltaMetricMetadata,
				IntDataPointSlice:          tt.fields.IntDataPointSlice,
			}
			if got := dps.At(tt.args.i); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("At() = %v, want %v", got, tt.want)
			}
		})
	}
}
