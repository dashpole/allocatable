package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/dashpole/allocatable/pkg/events/types"
	"k8s.io/api/core/v1"
)

const retryNumber = 2

func main() {
	fmt.Println("Getting Events")
	for i := 0; i < retryNumber; i++ {
		events, err := fetchEvents()
		if err == nil {
			fmt.Println(events.String())
			break
		}
		fmt.Printf("Error getting Events: %v\n", err)
		if i < retryNumber-1 {
			fmt.Printf("Retrying fetchEvents...")
		}
		time.Sleep(1 * time.Minute)
	}
	fmt.Println("Getting ClusterInfo")
	for i := 0; i < retryNumber; i++ {
		info, err := fetchClusterInfo()
		if err == nil {
			fmt.Println(info.String())
			break
		}
		fmt.Printf("Error getting ClusterInfo: %v\n", err)
		if i < retryNumber-1 {
			fmt.Printf("Retrying fetchClusterInfo...")
		}
		time.Sleep(1 * time.Minute)
	}
}

func fetchEvents() (types.DisruptiveEventList, error) {
	eventsBlob, err := exec.Command("kubectl", "get", "ev", "--all-namespaces=true", "-o", "json").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Error getting events: %v\n", err)
	}
	var eventList v1.EventList
	json.Unmarshal(eventsBlob, &eventList)

	return types.GetDisruptiveEventList(eventList.Items), nil
}

func fetchClusterInfo() (*types.ClusterInfo, error) {
	nodesBlob, err := exec.Command("kubectl", "get", "no", "-o", "json").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Error getting nodes: %v\n", err)
	}
	var nodeList v1.NodeList
	json.Unmarshal(nodesBlob, &nodeList)

	podsBlob, err := exec.Command("kubectl", "get", "po", "--all-namespaces=true", "-o", "json").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Error getting pods: %v\n", err)
	}
	var podList v1.PodList
	json.Unmarshal(podsBlob, &podList)

	return types.GetClusterInfo(podList.Items, nodeList.Items), nil
}
