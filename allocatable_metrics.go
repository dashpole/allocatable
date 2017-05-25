/*
Copyright 2016 The Kubernetes Authors.

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

package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	// "time"

	resourceapi "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/kubernetes/pkg/api/v1"
	"k8s.io/kubernetes/pkg/api/v1/resource"
)

type NodeAllocated struct {
	NodeName          string
	MemoryAllocatable resourceapi.Quantity
	CPUAllocatable    resourceapi.Quantity
	MemoryRequests    resourceapi.Quantity
	CPURequests       resourceapi.Quantity
}

const retryNumber = 2

// func main() {
// 	fmt.Printf("Getting Node Allocatable\n")
// 	for i := 0; i < retryNumber; i++ {
// 		nodeAllocatedList, err := fetchNodeAllocated()
// 		if err == nil {
// 			if len(nodeAllocatedList) == 0 {
// 				fmt.Printf("No Nodes Found\n")
// 			}
// 			for _, nodeAllocated := range nodeAllocatedList {
// 				fmt.Println(nodeAllocated.String())
// 			}
// 			return
// 		}
// 		fmt.Printf("Error getting Node Allocatable: %v\n", err)
// 		if i < retryNumber-1 {
// 			fmt.Printf("Retrying...")
// 		}
// 		time.Sleep(1 * time.Minute)
// 	}
// }

func fetchNodeAllocated() ([]NodeAllocated, error) {
	podsBlob, err := exec.Command("kubectl", "get", "pods", "--all-namespaces=true", "-o", "json").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Error getting pods: %v\n", err)
	}
	var podList v1.PodList
	json.Unmarshal(podsBlob, &podList)

	nodesBlob, err := exec.Command("kubectl", "get", "nodes", "-o", "json").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Error getting nodes: %v\n", err)
	}
	var nodeList v1.NodeList
	json.Unmarshal(nodesBlob, &nodeList)

	nodeAllocatedList, err := getNodeAllocatedList(podList.Items, nodeList.Items)
	if err != nil {
		return nil, fmt.Errorf("Error calculating node allocated: %v\n", err)
	}
	return nodeAllocatedList, nil
}

func getNodeAllocatedList(pods []v1.Pod, nodes []v1.Node) ([]NodeAllocated, error) {
	nodeAllocatedList := []NodeAllocated{}
	for _, node := range nodes {
		memoryRequests := resourceapi.NewQuantity(0, resourceapi.DecimalSI)
		cpuRequests := resourceapi.NewQuantity(0, resourceapi.DecimalSI)
		for _, pod := range pods {
			if pod.Spec.NodeName != node.Name {
				//skip if the pod is not on the current node
				continue
			}
			req, _, err := resource.PodRequestsAndLimits(&pod)
			if err != nil {
				return nil, fmt.Errorf("ERROR getting pod requests and limits: %v", err)
			}
			memoryRequests.Add(req[v1.ResourceMemory])
			cpuRequests.Add(req[v1.ResourceCPU])
		}
		nodeAllocatedList = append(nodeAllocatedList, NodeAllocated{
			NodeName:          node.Name,
			MemoryAllocatable: node.Status.Allocatable[v1.ResourceMemory],
			CPUAllocatable:    node.Status.Allocatable[v1.ResourceCPU],
			MemoryRequests:    *memoryRequests,
			CPURequests:       *cpuRequests,
		})
	}
	return nodeAllocatedList, nil
}

const allocatableTemplate = "NodeName: %s, Memory: %s / %s = %v%%, CPU: %s / %s = %v%%"

func (na *NodeAllocated) String() string {
	return fmt.Sprintf(allocatableTemplate, na.NodeName, na.MemoryRequests.String(), na.MemoryAllocatable.String(), na.GetMemoryPercent(), na.CPURequests.String(), na.CPUAllocatable.String(), na.GetCPUPercent())
}

func (na *NodeAllocated) GetMemoryPercent() int64 {
	return 100.0 * na.MemoryRequests.Value() / na.MemoryAllocatable.Value()
}

func (na *NodeAllocated) GetCPUPercent() int64 {
	return 100.0 * na.CPURequests.MilliValue() / na.CPUAllocatable.MilliValue()
}
