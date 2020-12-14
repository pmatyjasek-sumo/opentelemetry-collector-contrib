// Copyright 2020 OpenTelemetry Authors
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

package k8sprocessor

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/k8sconfig"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/k8sprocessor/kube"
)

const (
	k8sIPLabelName     string = "k8s.pod.ip"
	k8sPodUIDLabelName string = "k8s.pod.uid"
	podUIDLabelName    string = "pod_uid"
	clientIPLabelName  string = "ip"
	hostnameLabelName  string = "host.hostname"
)

type kubernetesprocessor struct {
	logger          *zap.Logger
	apiConfig       k8sconfig.APIConfig
	kc              kube.Client
	passthroughMode bool
	rules           kube.ExtractionRules
	filters         kube.Filters
	podAssociations kube.Associations
}

func (kp *kubernetesprocessor) initKubeClient(logger *zap.Logger, kubeClient kube.ClientProvider) error {
	if kubeClient == nil {
		kubeClient = kube.New
	}
	if !kp.passthroughMode {
		kc, err := kubeClient(logger, kp.apiConfig, kp.rules, kp.filters, kp.podAssociations, nil, nil)
		if err != nil {
			return err
		}
		kp.kc = kc
	}
	return nil
}

func (kp *kubernetesprocessor) Start(_ context.Context, _ component.Host) error {
	if !kp.passthroughMode {
		go kp.kc.Start()
	}
	return nil
}

func (kp *kubernetesprocessor) Shutdown(context.Context) error {
	if !kp.passthroughMode {
		kp.kc.Stop()
	}
	return nil
}

// ProcessTraces process traces and add k8s metadata using resource IP or incoming IP as pod origin.
func (kp *kubernetesprocessor) ProcessTraces(ctx context.Context, td pdata.Traces) (pdata.Traces, error) {
	rss := td.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		if rs.IsNil() {
			continue
		}

		kp.processResource(ctx, rs.Resource())
	}

	return td, nil
}

// ProcessMetrics process metrics and add k8s metadata using resource IP, hostname or incoming IP as pod origin.
func (kp *kubernetesprocessor) ProcessMetrics(ctx context.Context, md pdata.Metrics) (pdata.Metrics, error) {
	rm := md.ResourceMetrics()
	for i := 0; i < rm.Len(); i++ {
		ms := rm.At(i)
		if ms.IsNil() {
			continue
		}

		kp.processResource(ctx, ms.Resource())
	}

	return md, nil
}

// ProcessLogs process logs and add k8s metadata using resource IP, hostname or incoming IP as pod origin.
func (kp *kubernetesprocessor) ProcessLogs(ctx context.Context, ld pdata.Logs) (pdata.Logs, error) {
	rl := ld.ResourceLogs()
	for i := 0; i < rl.Len(); i++ {
		ls := rl.At(i)
		if ls.IsNil() {
			continue
		}

		kp.processResource(ctx, ls.Resource())
	}

	return ld, nil
}

func (kp *kubernetesprocessor) processResource(ctx context.Context, resource pdata.Resource) {

	podAttributes := k8sPodAssociationFromAttributes(ctx, resource.Attributes(), kp.podAssociations)
	if len(podAttributes) == 0 {
		return
	}
	attrsToAdd := make(map[string]string)

	for k, v := range podAttributes {
		if !kp.passthroughMode {
			attrsToAdd = kp.getAttributesForPod(v)
			for key, val := range attrsToAdd {
				resource.Attributes().InsertString(key, val)
			}
		}
		resource.Attributes().InsertString(k, v)
	}
}

func (kp *kubernetesprocessor) getAttributesForPod(identifier string) map[string]string {
	pod, ok := kp.kc.GetPod(identifier)
	if !ok {
		return nil
	}
	return pod.Attributes
}
