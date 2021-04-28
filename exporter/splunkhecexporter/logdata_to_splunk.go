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

package splunkhecexporter

import (
	"time"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/translator/conventions"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/splunk"
)

// Composite index of a log record in pdata.Logs.
type logIndex struct {
	// Index in orig list (i.e. root parent index).
	resource int
	// Index in InstrumentationLibraryLogs list (i.e. immediate parent index).
	library int
	// Index in Logs list (i.e. the log record index).
	record int
}

func (i *logIndex) zero() bool {
	return i.resource == 0 && i.library == 0 && i.record == 0
}

func mapLogRecordToSplunkEvent(res pdata.Resource, lr pdata.LogRecord, config *Config, logger *zap.Logger) *splunk.Event {
	host := unknownHostName
	source := config.Source
	sourcetype := config.SourceType
	index := config.Index
	fields := map[string]interface{}{}
	res.Attributes().ForEach(func(k string, v pdata.AttributeValue) {
		switch k {
		case conventions.AttributeHostName:
			host = v.StringVal()
			fields[k] = v.StringVal()
		case conventions.AttributeServiceName:
			source = v.StringVal()
			fields[k] = v.StringVal()
		case splunk.SourcetypeLabel:
			sourcetype = v.StringVal()
		case splunk.IndexLabel:
			index = v.StringVal()
		default:
			fields[k] = convertAttributeValue(v, logger)
		}
	})
	lr.Attributes().ForEach(func(k string, v pdata.AttributeValue) {
		switch k {
		case conventions.AttributeHostName:
			host = v.StringVal()
			fields[k] = v.StringVal()
		case conventions.AttributeServiceName:
			source = v.StringVal()
			fields[k] = v.StringVal()
		case splunk.SourcetypeLabel:
			sourcetype = v.StringVal()
		case splunk.IndexLabel:
			index = v.StringVal()
		default:
			fields[k] = convertAttributeValue(v, logger)
		}
	})

	eventValue := convertAttributeValue(lr.Body(), logger)
	return &splunk.Event{
		Time:       nanoTimestampToEpochMilliseconds(lr.Timestamp()),
		Host:       host,
		Source:     source,
		SourceType: sourcetype,
		Index:      index,
		Event:      eventValue,
		Fields:     fields,
	}
}

func convertAttributeValue(value pdata.AttributeValue, logger *zap.Logger) interface{} {
	switch value.Type() {
	case pdata.AttributeValueINT:
		return value.IntVal()
	case pdata.AttributeValueBOOL:
		return value.BoolVal()
	case pdata.AttributeValueDOUBLE:
		return value.DoubleVal()
	case pdata.AttributeValueSTRING:
		return value.StringVal()
	case pdata.AttributeValueMAP:
		values := map[string]interface{}{}
		value.MapVal().ForEach(func(k string, v pdata.AttributeValue) {
			values[k] = convertAttributeValue(v, logger)
		})
		return values
	case pdata.AttributeValueARRAY:
		arrayVal := value.ArrayVal()
		values := make([]interface{}, arrayVal.Len())
		for i := 0; i < arrayVal.Len(); i++ {
			values[i] = convertAttributeValue(arrayVal.At(i), logger)
		}
		return values
	case pdata.AttributeValueNULL:
		return nil
	default:
		logger.Debug("Unhandled value type", zap.String("type", value.Type().String()))
		return value
	}
}

// nanoTimestampToEpochMilliseconds transforms nanoseconds into <sec>.<ms>. For example, 1433188255.500 indicates 1433188255 seconds and 500 milliseconds after epoch.
func nanoTimestampToEpochMilliseconds(ts pdata.Timestamp) *float64 {
	duration := time.Duration(ts)
	if duration == 0 {
		// some telemetry sources send data with timestamps set to 0 by design, as their original target destinations
		// (i.e. before Open Telemetry) are setup with the know-how on how to consume them. In this case,
		// we want to omit the time field when sending data to the Splunk HEC so that the HEC adds a timestamp
		// at indexing time, which will be much more useful than a 0-epoch-time value.
		return nil
	}

	val := duration.Round(time.Millisecond).Seconds()
	return &val
}
