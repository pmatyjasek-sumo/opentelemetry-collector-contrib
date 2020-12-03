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
	"net"

	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/k8sprocessor/kube"
	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/translator/conventions"
)

func k8sPodAssociationFromAttributes(ctx context.Context, attrs pdata.AttributeMap, associations kube.Associations) map[string]string {
	var pod_ip, client_ip, context_ip string
	podAssociation := make(map[string]string)

	pod_ip = stringAttributeFromMap(attrs, k8sIPLabelName)
	client_ip = stringAttributeFromMap(attrs, clientIPLabelName)
	if c, ok := client.FromContext(ctx); ok {
		context_ip = c.IP
	}

	for _, asso := range associations.Associations {
		if asso.Name == "pod_uid" {
			uid := stringAttributeFromMap(attrs, asso.Name)
			if uid != "" {
				podAssociation[k8sPodUIDLabelName] = uid
			}
		}
		if _, ok := podAssociation[k8sIPLabelName]; !ok {
			if asso.Name == "k8s.pod.ip" && pod_ip != "" {
				podAssociation[k8sIPLabelName] = pod_ip
				continue
			}
			if asso.Name == "ip" && asso.From == "label" && client_ip != "" {
				podAssociation[k8sIPLabelName] = client_ip
				continue
			}
			if asso.Name == "ip" && asso.From == "connection" && context_ip != "" {
				podAssociation[k8sIPLabelName] = context_ip
				continue
			}
		}
		if asso.Name == "host.hostname" {
			hostname := stringAttributeFromMap(attrs, conventions.AttributeHostName)
			if net.ParseIP(hostname) != nil {
				podAssociation[k8sIPLabelName] = hostname
			}

		}
	}

	return podAssociation
}

func stringAttributeFromMap(attrs pdata.AttributeMap, key string) string {
	if val, ok := attrs.Get(key); ok {
		if val.Type() == pdata.AttributeValueSTRING {
			return val.StringVal()
		}
	}
	return ""
}
