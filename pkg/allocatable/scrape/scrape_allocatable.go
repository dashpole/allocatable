package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/dashpole/allocatable/pkg/allocatable/types"
)

const retryNumber = 2

func main() {
	fmt.Printf("Getting Node Allocatable\n")
	for i := 0; i < retryNumber; i++ {
		nodeAllocatedList, err := fetchNodeAllocated()
		if err == nil {
			if len(nodeAllocatedList) == 0 {
				fmt.Printf("No Nodes Found\n")
			}
			for _, nodeAllocated := range nodeAllocatedList {
				fmt.Println(nodeAllocated.String())
			}
			return
		}
		fmt.Printf("Error getting Node Allocatable: %v\n", err)
		if i < retryNumber-1 {
			fmt.Printf("Retrying...")
		}
		time.Sleep(1 * time.Minute)
	}
}

func fetchNodeAllocated() ([]types.NodeAllocated, error) {
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

func getNodeAllocatedList(pods []v1.Pod, nodes []v1.Node) ([]types.NodeAllocated, error) {
	nodeAllocatedList := []types.NodeAllocated{}
	for _, node := range nodes {
		memoryRequests := resource.NewQuantity(0, resource.DecimalSI)
		cpuRequests := resource.NewQuantity(0, resource.DecimalSI)
		for _, pod := range pods {
			if pod.Spec.NodeName != node.Name {
				//skip if the pod is not on the current node
				continue
			}
			req, _ := PodRequestsAndLimits(&pod)
			memoryRequests.Add(req[v1.ResourceMemory])
			cpuRequests.Add(req[v1.ResourceCPU])
		}
		nodeAllocatedList = append(nodeAllocatedList, types.NodeAllocated{
			NodeName:          node.Name,
			MemoryAllocatable: node.Status.Allocatable[v1.ResourceMemory],
			CPUAllocatable:    node.Status.Allocatable[v1.ResourceCPU],
			MemoryRequests:    *memoryRequests,
			CPURequests:       *cpuRequests,
		})
	}
	return nodeAllocatedList, nil
}

// PodRequestsAndLimits returns a dictionary of all defined resources summed up for all
// containers of the pod.
func PodRequestsAndLimits(pod *v1.Pod) (reqs map[v1.ResourceName]resource.Quantity, limits map[v1.ResourceName]resource.Quantity) {
	reqs, limits = map[v1.ResourceName]resource.Quantity{}, map[v1.ResourceName]resource.Quantity{}
	for _, container := range pod.Spec.Containers {
		for name, quantity := range container.Resources.Requests {
			if value, ok := reqs[name]; !ok {
				reqs[name] = *quantity.Copy()
			} else {
				value.Add(quantity)
				reqs[name] = value
			}
		}
		for name, quantity := range container.Resources.Limits {
			if value, ok := limits[name]; !ok {
				limits[name] = *quantity.Copy()
			} else {
				value.Add(quantity)
				limits[name] = value
			}
		}
	}
	// init containers define the minimum of any resource
	for _, container := range pod.Spec.InitContainers {
		for name, quantity := range container.Resources.Requests {
			value, ok := reqs[name]
			if !ok {
				reqs[name] = *quantity.Copy()
				continue
			}
			if quantity.Cmp(value) > 0 {
				reqs[name] = *quantity.Copy()
			}
		}
		for name, quantity := range container.Resources.Limits {
			value, ok := limits[name]
			if !ok {
				limits[name] = *quantity.Copy()
				continue
			}
			if quantity.Cmp(value) > 0 {
				limits[name] = *quantity.Copy()
			}
		}
	}
	return
}
