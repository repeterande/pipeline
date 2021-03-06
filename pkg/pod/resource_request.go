/*
Copyright 2019 The Tekton Authors

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

package pod

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var emptyResourceQuantity = resource.Quantity{}
var zeroQty = resource.MustParse("0")

func allZeroQty() corev1.ResourceList {
	return corev1.ResourceList{
		corev1.ResourceCPU:              zeroQty,
		corev1.ResourceMemory:           zeroQty,
		corev1.ResourceEphemeralStorage: zeroQty,
	}
}

func resolveResourceRequests(containers []corev1.Container, limitRange *corev1.LimitRange) []corev1.Container {
	max := allZeroQty()
	resourceNames := []corev1.ResourceName{corev1.ResourceCPU, corev1.ResourceMemory, corev1.ResourceEphemeralStorage}
	maxIndicesByResource := make(map[corev1.ResourceName]int, len(resourceNames))
	for _, resourceName := range resourceNames {
		maxIndicesByResource[resourceName] = -1
	}

	// Find max resource requests and associated list indices for
	// containers for CPU, memory, and ephemeral storage resources
	for i, c := range containers {
		for k, v := range c.Resources.Requests {
			if v.Cmp(max[k]) > 0 {
				maxIndicesByResource[k] = i
				max[k] = v
			}
		}
	}

	// Get limitrange minimum for container requests so they won't
	// be zeroed out if minimum is specified in namespace
	var limitRangeItems []corev1.LimitRangeItem
	if limitRange != nil {
		limitRangeItems = limitRange.Spec.Limits
	}
	min := allZeroQty()
	for _, limitRangeItem := range limitRangeItems {
		if limitRangeItem.Type == corev1.LimitTypeContainer {
			if limitRangeItem.Min != nil {
				min = limitRangeItem.Min
			}
			break
		}
	}

	// Use zeroQty if request value is not set for min
	if min[corev1.ResourceCPU] == emptyResourceQuantity {
		min[corev1.ResourceCPU] = zeroQty
	}
	if min[corev1.ResourceMemory] == emptyResourceQuantity {
		min[corev1.ResourceMemory] = zeroQty
	}
	if min[corev1.ResourceEphemeralStorage] == emptyResourceQuantity {
		min[corev1.ResourceEphemeralStorage] = zeroQty
	}

	// Set all non max resource requests to 0. Leave max request at index
	// originally defined to account for limit of step.
	for i := range containers {
		if containers[i].Resources.Requests == nil {
			containers[i].Resources.Requests = min
			continue
		}
		for _, resourceName := range resourceNames {
			if maxIndicesByResource[resourceName] != i {
				containers[i].Resources.Requests[resourceName] = min[resourceName]
			}
		}
	}

	return containers
}
