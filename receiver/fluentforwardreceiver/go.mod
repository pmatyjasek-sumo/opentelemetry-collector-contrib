module github.com/open-telemetry/opentelemetry-collector-contrib/receiver/fluentforwardreceiver

go 1.15

require (
	github.com/stretchr/testify v1.7.0
	github.com/tinylib/msgp v1.1.5
	go.opencensus.io v0.23.0
	go.opentelemetry.io/collector v0.25.1-0.20210421230708-d10b842f49eb
	go.uber.org/zap v1.16.0
)
// WIP update for otelcol changes
replace go.opentelemetry.io/collector => github.com/pmatyjasek-sumo/opentelemetry-collector v0.25.1-0.20210428081312-72ef9d6ccfe5
