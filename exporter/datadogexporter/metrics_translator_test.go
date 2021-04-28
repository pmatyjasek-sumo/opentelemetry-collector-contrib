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

package datadogexporter

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/consumer/pdata"
	"gopkg.in/zorkian/go-datadog-api.v2"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/config"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/metadata"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/metrics"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/common/ttlmap"
)

func TestMetricValue(t *testing.T) {
	var (
		name  string   = "name"
		value float64  = math.Pi
		ts    uint64   = uint64(time.Now().UnixNano())
		tags  []string = []string{"tool:opentelemetry", "version:0.1.0"}
	)

	metric := metrics.NewGauge(name, ts, value, tags)
	assert.Equal(t, metrics.Gauge, metric.GetType())
	assert.Equal(t, tags, metric.Tags)
}

func TestGetTags(t *testing.T) {
	labels := pdata.NewStringMap()
	labels.InitFromMap(map[string]string{
		"key1": "val1",
		"key2": "val2",
		"key3": "",
	})

	assert.ElementsMatch(t,
		getTags(labels),
		[...]string{"key1:val1", "key2:val2", "key3:n/a"},
	)
}

func TestIsCumulativeMonotonic(t *testing.T) {
	// Some of these examples are from the hostmetrics receiver
	// and reflect the semantic meaning of the metrics there.
	//
	// If the receiver changes these examples should be added here too

	{ // IntSum: Cumulative but not monotonic
		metric := pdata.NewMetric()
		metric.SetName("system.filesystem.usage")
		metric.SetDescription("Filesystem bytes used.")
		metric.SetUnit("bytes")
		metric.SetDataType(pdata.MetricDataTypeIntSum)
		sum := metric.IntSum()
		sum.SetIsMonotonic(false)
		sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)

		assert.False(t, isCumulativeMonotonic(metric))
	}

	{ // IntSum: Cumulative and monotonic
		metric := pdata.NewMetric()
		metric.SetName("system.network.packets")
		metric.SetDescription("The number of packets transferred.")
		metric.SetUnit("1")
		metric.SetDataType(pdata.MetricDataTypeIntSum)
		sum := metric.IntSum()
		sum.SetIsMonotonic(true)
		sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)

		assert.True(t, isCumulativeMonotonic(metric))
	}

	{ // DoubleSumL Cumulative and monotonic
		metric := pdata.NewMetric()
		metric.SetName("metric.example")
		metric.SetDataType(pdata.MetricDataTypeDoubleSum)
		sum := metric.DoubleSum()
		sum.SetIsMonotonic(true)
		sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)

		assert.True(t, isCumulativeMonotonic(metric))
	}

	{ // Not IntSum
		metric := pdata.NewMetric()
		metric.SetName("system.cpu.load_average.1m")
		metric.SetDescription("Average CPU Load over 1 minute.")
		metric.SetUnit("1")
		metric.SetDataType(pdata.MetricDataTypeDoubleGauge)

		assert.False(t, isCumulativeMonotonic(metric))
	}
}

func TestMetricDimensionsToMapKey(t *testing.T) {
	metricName := "metric.name"
	noTags := metricDimensionsToMapKey(metricName, []string{})
	someTags := metricDimensionsToMapKey(metricName, []string{"key1:val1", "key2:val2"})
	sameTags := metricDimensionsToMapKey(metricName, []string{"key2:val2", "key1:val1"})
	diffTags := metricDimensionsToMapKey(metricName, []string{"key3:val3"})

	assert.NotEqual(t, noTags, someTags)
	assert.NotEqual(t, someTags, diffTags)
	assert.Equal(t, someTags, sameTags)
}

func TestMapIntMetrics(t *testing.T) {
	ts := pdata.TimestampFromTime(time.Now())
	slice := pdata.NewIntDataPointSlice()
	slice.Resize(1)
	point := slice.At(0)
	point.SetValue(17)
	point.SetTimestamp(ts)

	assert.ElementsMatch(t,
		mapIntMetrics("int64.test", slice, []string{}),
		[]datadog.Metric{metrics.NewGauge("int64.test", uint64(ts), 17, []string{})},
	)

	// With attribute tags
	assert.ElementsMatch(t,
		mapIntMetrics("int64.test", slice, []string{"attribute_tag:attribute_value"}),
		[]datadog.Metric{metrics.NewGauge("int64.test", uint64(ts), 17, []string{"attribute_tag:attribute_value"})},
	)
}

func TestMapDoubleMetrics(t *testing.T) {
	ts := pdata.TimestampFromTime(time.Now())
	slice := pdata.NewDoubleDataPointSlice()
	slice.Resize(1)
	point := slice.At(0)
	point.SetValue(math.Pi)
	point.SetTimestamp(ts)

	assert.ElementsMatch(t,
		mapDoubleMetrics("float64.test", slice, []string{}),
		[]datadog.Metric{metrics.NewGauge("float64.test", uint64(ts), math.Pi, []string{})},
	)

	// With attribute tags
	assert.ElementsMatch(t,
		mapDoubleMetrics("float64.test", slice, []string{"attribute_tag:attribute_value"}),
		[]datadog.Metric{metrics.NewGauge("float64.test", uint64(ts), math.Pi, []string{"attribute_tag:attribute_value"})},
	)
}

func newTTLMap() *ttlmap.TTLMap {
	// don't start the sweeping goroutine
	// since it is not needed
	return ttlmap.New(1800, 3600)
}

func seconds(i int) pdata.Timestamp {
	return pdata.TimestampFromTime(time.Unix(int64(i), 0))
}

func TestMapIntMonotonicMetrics(t *testing.T) {
	// Create list of values
	deltas := []int64{1, 2, 200, 3, 7, 0}
	cumulative := make([]int64, len(deltas)+1)
	cumulative[0] = 0
	for i := 1; i < len(cumulative); i++ {
		cumulative[i] = cumulative[i-1] + deltas[i-1]
	}

	//Map to OpenTelemetry format
	slice := pdata.NewIntDataPointSlice()
	slice.Resize(len(cumulative))
	for i, val := range cumulative {
		point := slice.At(i)
		point.SetValue(val)
		point.SetTimestamp(seconds(i))
	}

	// Map to Datadog format
	metricName := "metric.example"
	expected := make([]datadog.Metric, len(deltas))
	for i, val := range deltas {
		expected[i] = metrics.NewCount(metricName, uint64(seconds(i+1)), float64(val), []string{})
	}

	prevPts := newTTLMap()
	output := mapIntMonotonicMetrics(metricName, prevPts, slice, []string{})

	assert.ElementsMatch(t, output, expected)
}

func TestMapIntMonotonicDifferentDimensions(t *testing.T) {
	metricName := "metric.example"
	slice := pdata.NewIntDataPointSlice()
	slice.Resize(6)

	// No tags
	point := slice.At(0)
	point.SetTimestamp(seconds(0))

	point = slice.At(1)
	point.SetValue(20)
	point.SetTimestamp(seconds(1))

	// One tag: valA
	point = slice.At(2)
	point.SetTimestamp(seconds(0))
	point.LabelsMap().Insert("key1", "valA")

	point = slice.At(3)
	point.SetValue(30)
	point.SetTimestamp(seconds(1))
	point.LabelsMap().Insert("key1", "valA")

	// same tag: valB
	point = slice.At(4)
	point.SetTimestamp(seconds(0))
	point.LabelsMap().Insert("key1", "valB")

	point = slice.At(5)
	point.SetValue(40)
	point.SetTimestamp(seconds(1))
	point.LabelsMap().Insert("key1", "valB")

	prevPts := newTTLMap()

	assert.ElementsMatch(t,
		mapIntMonotonicMetrics(metricName, prevPts, slice, []string{}),
		[]datadog.Metric{
			metrics.NewCount(metricName, uint64(seconds(1)), 20, []string{}),
			metrics.NewCount(metricName, uint64(seconds(1)), 30, []string{"key1:valA"}),
			metrics.NewCount(metricName, uint64(seconds(1)), 40, []string{"key1:valB"}),
		},
	)
}

func TestMapIntMonotonicWithReboot(t *testing.T) {
	values := []int64{0, 30, 0, 20}
	metricName := "metric.example"
	slice := pdata.NewIntDataPointSlice()
	slice.Resize(len(values))

	for i, val := range values {
		point := slice.At(i)
		point.SetTimestamp(seconds(i))
		point.SetValue(val)
	}

	prevPts := newTTLMap()
	assert.ElementsMatch(t,
		mapIntMonotonicMetrics(metricName, prevPts, slice, []string{}),
		[]datadog.Metric{
			metrics.NewCount(metricName, uint64(seconds(1)), 30, []string{}),
			metrics.NewCount(metricName, uint64(seconds(3)), 20, []string{}),
		},
	)
}

func TestMapIntMonotonicOutOfOrder(t *testing.T) {
	stamps := []int{1, 0, 2, 3}
	values := []int64{0, 1, 2, 3}

	metricName := "metric.example"
	slice := pdata.NewIntDataPointSlice()
	slice.Resize(len(values))

	for i, val := range values {
		point := slice.At(i)
		point.SetTimestamp(seconds(stamps[i]))
		point.SetValue(val)
	}

	prevPts := newTTLMap()
	assert.ElementsMatch(t,
		mapIntMonotonicMetrics(metricName, prevPts, slice, []string{}),
		[]datadog.Metric{
			metrics.NewCount(metricName, uint64(seconds(2)), 2, []string{}),
			metrics.NewCount(metricName, uint64(seconds(3)), 1, []string{}),
		},
	)
}

func TestMapDoubleMonotonicMetrics(t *testing.T) {
	deltas := []float64{1, 2, 200, 3, 7, 0}
	cumulative := make([]float64, len(deltas)+1)
	cumulative[0] = 0
	for i := 1; i < len(cumulative); i++ {
		cumulative[i] = cumulative[i-1] + deltas[i-1]
	}

	//Map to OpenTelemetry format
	slice := pdata.NewDoubleDataPointSlice()
	slice.Resize(len(cumulative))
	for i, val := range cumulative {
		point := slice.At(i)
		point.SetValue(val)
		point.SetTimestamp(seconds(i))
	}

	// Map to Datadog format
	metricName := "metric.example"
	expected := make([]datadog.Metric, len(deltas))
	for i, val := range deltas {
		expected[i] = metrics.NewCount(metricName, uint64(seconds(i+1)), val, []string{})
	}

	prevPts := newTTLMap()
	output := mapDoubleMonotonicMetrics(metricName, prevPts, slice, []string{})

	assert.ElementsMatch(t, expected, output)
}

func TestMapDoubleMonotonicDifferentDimensions(t *testing.T) {
	metricName := "metric.example"
	slice := pdata.NewDoubleDataPointSlice()
	slice.Resize(6)

	// No tags
	point := slice.At(0)
	point.SetTimestamp(seconds(0))

	point = slice.At(1)
	point.SetValue(20)
	point.SetTimestamp(seconds(1))

	// One tag: valA
	point = slice.At(2)
	point.SetTimestamp(seconds(0))
	point.LabelsMap().Insert("key1", "valA")

	point = slice.At(3)
	point.SetValue(30)
	point.SetTimestamp(seconds(1))
	point.LabelsMap().Insert("key1", "valA")

	// one tag: valB
	point = slice.At(4)
	point.SetTimestamp(seconds(0))
	point.LabelsMap().Insert("key1", "valB")

	point = slice.At(5)
	point.SetValue(40)
	point.SetTimestamp(seconds(1))
	point.LabelsMap().Insert("key1", "valB")

	prevPts := newTTLMap()

	assert.ElementsMatch(t,
		mapDoubleMonotonicMetrics(metricName, prevPts, slice, []string{}),
		[]datadog.Metric{
			metrics.NewCount(metricName, uint64(seconds(1)), 20, []string{}),
			metrics.NewCount(metricName, uint64(seconds(1)), 30, []string{"key1:valA"}),
			metrics.NewCount(metricName, uint64(seconds(1)), 40, []string{"key1:valB"}),
		},
	)
}

func TestMapDoubleMonotonicWithReboot(t *testing.T) {
	values := []float64{0, 30, 0, 20}
	metricName := "metric.example"
	slice := pdata.NewDoubleDataPointSlice()
	slice.Resize(len(values))

	for i, val := range values {
		point := slice.At(i)
		point.SetTimestamp(seconds(2 * i))
		point.SetValue(val)
	}

	prevPts := newTTLMap()
	assert.ElementsMatch(t,
		mapDoubleMonotonicMetrics(metricName, prevPts, slice, []string{}),
		[]datadog.Metric{
			metrics.NewCount(metricName, uint64(seconds(2)), 30, []string{}),
			metrics.NewCount(metricName, uint64(seconds(6)), 20, []string{}),
		},
	)
}

func TestMapDoubleMonotonicOutOfOrder(t *testing.T) {
	stamps := []int{1, 0, 2, 3}
	values := []float64{0, 1, 2, 3}

	metricName := "metric.example"
	slice := pdata.NewDoubleDataPointSlice()
	slice.Resize(len(values))

	for i, val := range values {
		point := slice.At(i)
		point.SetTimestamp(seconds(stamps[i]))
		point.SetValue(val)
	}

	prevPts := newTTLMap()
	assert.ElementsMatch(t,
		mapDoubleMonotonicMetrics(metricName, prevPts, slice, []string{}),
		[]datadog.Metric{
			metrics.NewCount(metricName, uint64(seconds(2)), 2, []string{}),
			metrics.NewCount(metricName, uint64(seconds(3)), 1, []string{}),
		},
	)
}

func TestMapIntHistogramMetrics(t *testing.T) {
	ts := pdata.TimestampFromTime(time.Now())
	slice := pdata.NewIntHistogramDataPointSlice()
	slice.Resize(1)
	point := slice.At(0)
	point.SetCount(20)
	point.SetSum(200)
	point.SetBucketCounts([]uint64{2, 18})
	point.SetTimestamp(ts)

	noBuckets := []datadog.Metric{
		metrics.NewGauge("intHist.test.count", uint64(ts), 20, []string{}),
		metrics.NewGauge("intHist.test.sum", uint64(ts), 200, []string{}),
	}

	buckets := []datadog.Metric{
		metrics.NewGauge("intHist.test.count_per_bucket", uint64(ts), 2, []string{"bucket_idx:0"}),
		metrics.NewGauge("intHist.test.count_per_bucket", uint64(ts), 18, []string{"bucket_idx:1"}),
	}

	assert.ElementsMatch(t,
		mapIntHistogramMetrics("intHist.test", slice, false, []string{}), // No buckets
		noBuckets,
	)

	assert.ElementsMatch(t,
		mapIntHistogramMetrics("intHist.test", slice, true, []string{}), // buckets
		append(noBuckets, buckets...),
	)

	// With attribute tags
	noBucketsAttributeTags := []datadog.Metric{
		metrics.NewGauge("intHist.test.count", uint64(ts), 20, []string{"attribute_tag:attribute_value"}),
		metrics.NewGauge("intHist.test.sum", uint64(ts), 200, []string{"attribute_tag:attribute_value"}),
	}

	bucketsAttributeTags := []datadog.Metric{
		metrics.NewGauge("intHist.test.count_per_bucket", uint64(ts), 2, []string{"attribute_tag:attribute_value", "bucket_idx:0"}),
		metrics.NewGauge("intHist.test.count_per_bucket", uint64(ts), 18, []string{"attribute_tag:attribute_value", "bucket_idx:1"}),
	}

	assert.ElementsMatch(t,
		mapIntHistogramMetrics("intHist.test", slice, false, []string{"attribute_tag:attribute_value"}), // No buckets
		noBucketsAttributeTags,
	)

	assert.ElementsMatch(t,
		mapIntHistogramMetrics("intHist.test", slice, true, []string{"attribute_tag:attribute_value"}), // buckets
		append(noBucketsAttributeTags, bucketsAttributeTags...),
	)
}

func TestMapDoubleHistogramMetrics(t *testing.T) {
	ts := pdata.TimestampFromTime(time.Now())
	slice := pdata.NewDoubleHistogramDataPointSlice()
	slice.Resize(1)
	point := slice.At(0)
	point.SetCount(20)
	point.SetSum(math.Pi)
	point.SetBucketCounts([]uint64{2, 18})
	point.SetTimestamp(ts)

	noBuckets := []datadog.Metric{
		metrics.NewGauge("doubleHist.test.count", uint64(ts), 20, []string{}),
		metrics.NewGauge("doubleHist.test.sum", uint64(ts), math.Pi, []string{}),
	}

	buckets := []datadog.Metric{
		metrics.NewGauge("doubleHist.test.count_per_bucket", uint64(ts), 2, []string{"bucket_idx:0"}),
		metrics.NewGauge("doubleHist.test.count_per_bucket", uint64(ts), 18, []string{"bucket_idx:1"}),
	}

	assert.ElementsMatch(t,
		mapDoubleHistogramMetrics("doubleHist.test", slice, false, []string{}), // No buckets
		noBuckets,
	)

	assert.ElementsMatch(t,
		mapDoubleHistogramMetrics("doubleHist.test", slice, true, []string{}), // buckets
		append(noBuckets, buckets...),
	)

	// With attribute tags
	noBucketsAttributeTags := []datadog.Metric{
		metrics.NewGauge("doubleHist.test.count", uint64(ts), 20, []string{"attribute_tag:attribute_value"}),
		metrics.NewGauge("doubleHist.test.sum", uint64(ts), math.Pi, []string{"attribute_tag:attribute_value"}),
	}

	bucketsAttributeTags := []datadog.Metric{
		metrics.NewGauge("doubleHist.test.count_per_bucket", uint64(ts), 2, []string{"attribute_tag:attribute_value", "bucket_idx:0"}),
		metrics.NewGauge("doubleHist.test.count_per_bucket", uint64(ts), 18, []string{"attribute_tag:attribute_value", "bucket_idx:1"}),
	}

	assert.ElementsMatch(t,
		mapDoubleHistogramMetrics("doubleHist.test", slice, false, []string{"attribute_tag:attribute_value"}), // No buckets
		noBucketsAttributeTags,
	)

	assert.ElementsMatch(t,
		mapDoubleHistogramMetrics("doubleHist.test", slice, true, []string{"attribute_tag:attribute_value"}), // buckets
		append(noBucketsAttributeTags, bucketsAttributeTags...),
	)
}

func TestRunningMetrics(t *testing.T) {
	ms := pdata.NewMetrics()
	rms := ms.ResourceMetrics()
	rms.Resize(4)

	rm := rms.At(0)
	resAttrs := rm.Resource().Attributes()
	resAttrs.Insert(metadata.AttributeDatadogHostname, pdata.NewAttributeValueString("resource-hostname-1"))

	rm = rms.At(1)
	resAttrs = rm.Resource().Attributes()
	resAttrs.Insert(metadata.AttributeDatadogHostname, pdata.NewAttributeValueString("resource-hostname-1"))

	rm = rms.At(2)
	resAttrs = rm.Resource().Attributes()
	resAttrs.Insert(metadata.AttributeDatadogHostname, pdata.NewAttributeValueString("resource-hostname-2"))

	cfg := config.MetricsConfig{}
	prevPts := newTTLMap()

	series, _ := mapMetrics(cfg, prevPts, ms)

	runningHostnames := []string{}
	noHostname := 0

	for _, metric := range series {
		if *metric.Metric == "datadog_exporter.metrics.running" {
			if metric.Host != nil {
				runningHostnames = append(runningHostnames, *metric.Host)
			} else {
				noHostname++
			}
		}
	}

	assert.Equal(t, noHostname, 1)
	assert.ElementsMatch(t,
		runningHostnames,
		[]string{"resource-hostname-1", "resource-hostname-1", "resource-hostname-2"},
	)

}

const (
	testHostname = "res-hostname"
)

func createTestMetrics() pdata.Metrics {
	md := pdata.NewMetrics()
	rms := md.ResourceMetrics()
	rms.Resize(1)

	rm := rms.At(0)

	attrs := rm.Resource().Attributes()
	attrs.InsertString(metadata.AttributeDatadogHostname, testHostname)
	ilms := rm.InstrumentationLibraryMetrics()
	ilms.Resize(1)

	metricsArray := ilms.At(0).Metrics()
	metricsArray.Resize(9) // first one is TypeNone to test that it's ignored

	// IntGauge
	met := metricsArray.At(1)
	met.SetName("int.gauge")
	met.SetDataType(pdata.MetricDataTypeIntGauge)
	dpsInt := met.IntGauge().DataPoints()
	dpsInt.Resize(1)
	dpInt := dpsInt.At(0)
	dpInt.SetTimestamp(seconds(0))
	dpInt.SetValue(1)

	// DoubleGauge
	met = metricsArray.At(2)
	met.SetName("double.gauge")
	met.SetDataType(pdata.MetricDataTypeDoubleGauge)
	dpsDouble := met.DoubleGauge().DataPoints()
	dpsDouble.Resize(1)
	dpDouble := dpsDouble.At(0)
	dpDouble.SetTimestamp(seconds(0))
	dpDouble.SetValue(math.Pi)

	// IntSum
	met = metricsArray.At(3)
	met.SetName("int.sum")
	met.SetDataType(pdata.MetricDataTypeIntSum)
	dpsInt = met.IntSum().DataPoints()
	dpsInt.Resize(1)
	dpInt = dpsInt.At(0)
	dpInt.SetTimestamp(seconds(0))
	dpInt.SetValue(2)

	// DoubleSum
	met = metricsArray.At(4)
	met.SetName("double.sum")
	met.SetDataType(pdata.MetricDataTypeDoubleSum)
	dpsDouble = met.DoubleSum().DataPoints()
	dpsDouble.Resize(1)
	dpDouble = dpsDouble.At(0)
	dpDouble.SetTimestamp(seconds(0))
	dpDouble.SetValue(math.E)

	// IntHistogram
	met = metricsArray.At(5)
	met.SetName("int.histogram")
	met.SetDataType(pdata.MetricDataTypeIntHistogram)
	dpsIntHist := met.IntHistogram().DataPoints()
	dpsIntHist.Resize(1)
	dpIntHist := dpsIntHist.At(0)
	dpIntHist.SetCount(20)
	dpIntHist.SetSum(100)
	dpIntHist.SetBucketCounts([]uint64{2, 18})
	dpIntHist.SetTimestamp(seconds(0))

	// DoubleHistogram
	met = metricsArray.At(6)
	met.SetName("double.histogram")
	met.SetDataType(pdata.MetricDataTypeDoubleHistogram)
	dpsDoubleHist := met.DoubleHistogram().DataPoints()
	dpsDoubleHist.Resize(1)
	dpDoubleHist := dpsDoubleHist.At(0)
	dpDoubleHist.SetCount(20)
	dpDoubleHist.SetSum(math.Phi)
	dpDoubleHist.SetBucketCounts([]uint64{2, 18})
	dpDoubleHist.SetTimestamp(seconds(0))

	// Int Sum (cumulative)
	met = metricsArray.At(7)
	met.SetName("int.cumulative.sum")
	met.SetDataType(pdata.MetricDataTypeIntSum)
	met.IntSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	met.IntSum().SetIsMonotonic(true)
	dpsInt = met.IntSum().DataPoints()
	dpsInt.Resize(2)
	dpInt = dpsInt.At(0)
	dpInt.SetTimestamp(seconds(0))
	dpInt.SetValue(4)
	dpInt = dpsInt.At(1)
	dpInt.SetTimestamp(seconds(2))
	dpInt.SetValue(7)

	// Double Sum (cumulative)
	met = metricsArray.At(8)
	met.SetName("double.cumulative.sum")
	met.SetDataType(pdata.MetricDataTypeDoubleSum)
	met.DoubleSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	met.DoubleSum().SetIsMonotonic(true)
	dpsDouble = met.DoubleSum().DataPoints()
	dpsDouble.Resize(2)
	dpDouble = dpsDouble.At(0)
	dpDouble.SetTimestamp(seconds(0))
	dpDouble.SetValue(4)
	dpDouble = dpsDouble.At(1)
	dpDouble.SetTimestamp(seconds(2))
	dpDouble.SetValue(4 + math.Pi)

	return md
}

func removeRunningMetrics(series []datadog.Metric) []datadog.Metric {
	filtered := []datadog.Metric{}
	for _, m := range series {
		if m.GetMetric() != "datadog_exporter.metrics.running" {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

func testGauge(name string, val float64) datadog.Metric {
	m := metrics.NewGauge(name, 0, val, []string{})
	m.SetHost(testHostname)
	return m
}

func testCount(name string, val float64) datadog.Metric {
	m := metrics.NewCount(name, 2*1e9, val, []string{})
	m.SetHost(testHostname)
	return m
}

func TestMapMetrics(t *testing.T) {
	md := createTestMetrics()
	cfg := config.MetricsConfig{SendMonotonic: true}
	series, dropped := mapMetrics(cfg, newTTLMap(), md)
	assert.Equal(t, dropped, 0)
	filtered := removeRunningMetrics(series)
	assert.ElementsMatch(t, filtered, []datadog.Metric{
		testGauge("int.gauge", 1),
		testGauge("double.gauge", math.Pi),
		testGauge("int.sum", 2),
		testGauge("double.sum", math.E),
		testGauge("int.histogram.sum", 100),
		testGauge("int.histogram.count", 20),
		testGauge("double.histogram.sum", math.Phi),
		testGauge("double.histogram.count", 20),
		testCount("int.cumulative.sum", 3),
		testCount("double.cumulative.sum", math.Pi),
	})
}
