module github.com/open-telemetry/opentelemetry-collector-contrib/receiver/simpleprometheusreceiver

go 1.15

require (
	github.com/armon/go-metrics v0.3.3 // indirect
	github.com/gogo/googleapis v1.3.0 // indirect
	github.com/hashicorp/go-immutable-radix v1.2.0 // indirect
	github.com/hashicorp/go-msgpack v0.5.5 // indirect
	github.com/mattn/go-colorable v0.1.7 // indirect
	github.com/onsi/ginkgo v1.14.1 // indirect
	github.com/onsi/gomega v1.10.2 // indirect
	github.com/pelletier/go-toml v1.8.0 // indirect
	github.com/prometheus/common v0.23.0
	github.com/prometheus/prometheus v1.8.2-0.20210430082741-2a4b8e12bbf2
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/collector v0.25.1-0.20210504213736-b8824d020718
	go.uber.org/zap v1.16.0
	gopkg.in/ini.v1 v1.57.0 // indirect
	k8s.io/client-go v0.21.0
)
// WIP update for otelcol changes
replace go.opentelemetry.io/collector => github.com/pmatyjasek-sumo/opentelemetry-collector v0.25.1-0.20210428081312-72ef9d6ccfe5
