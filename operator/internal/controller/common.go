/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"sort"

	corev1alpha1 "github.com/kaleido-io/paladin/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func mergeServicePorts(svcSpec *corev1.ServiceSpec, requiredPorts []corev1.ServicePort) {
	portsByName := map[string]*corev1.ServicePort{}
	for _, providedPort := range svcSpec.Ports {
		tmpPort := providedPort
		portsByName[providedPort.Name] = &tmpPort
	}
	for _, requiredPort := range requiredPorts {
		providedPort, isProvided := portsByName[requiredPort.Name]
		if !isProvided {
			// Just use our definition
			tmpPort := requiredPort
			portsByName[requiredPort.Name] = &tmpPort
		} else {
			// We own the target port number and protocol always
			providedPort.TargetPort = requiredPort.TargetPort
			providedPort.Protocol = requiredPort.Protocol
			// Port can be overridden
			if providedPort.Port == 0 {
				providedPort.Port = requiredPort.Port
			}
		}
	}
	portNames := make([]string, 0, len(portsByName))
	for portName := range portsByName {
		portNames = append(portNames, portName)
	}
	// Need to sort by name to give deterministic behavior
	sort.Strings(portNames)
	svcSpec.Ports = make([]corev1.ServicePort, len(portNames))
	for i, portName := range portNames {
		svcSpec.Ports[i] = *portsByName[portName]
	}
}

func setCondition(
	conditions *[]metav1.Condition,
	conditionType corev1alpha1.ConditionType,
	status metav1.ConditionStatus,
	reason corev1alpha1.ConditionReason,
	message string,
) {
	condition := metav1.Condition{
		Type:               string(conditionType),
		Status:             status,
		Reason:             string(reason),
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}

	// Update or append the condition
	meta.SetStatusCondition(conditions, condition)
}
