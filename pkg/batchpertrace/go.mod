module github.com/open-telemetry/opentelemetry-collector-contrib/pkg/batchpertrace

go 1.14

require (
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/batchpersignal v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/collector v0.22.1-0.20210313012550-03904de3dd61
)

replace github.com/open-telemetry/opentelemetry-collector-contrib/pkg/batchpersignal => ../batchpersignal
