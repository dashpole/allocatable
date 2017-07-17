package types

import (
	"fmt"
	"strconv"
	"strings"

	"k8s.io/api/core/v1"
	resourceapi "k8s.io/apimachinery/pkg/api/resource"
)

func (c ClusterStats) ToSlice() []string {
	return []string{
		strconv.Itoa(int(c.NumNodes)),
		strconv.Itoa(int(c.ClusterCPU)),
		strconv.Itoa(int(c.ClusterMemory)),
		strconv.Itoa(int(c.ClusterCPUReserved)),
		strconv.Itoa(int(c.ClusterMemoryReserved)),
		strconv.Itoa(int(c.TotalPerNodeCPUOverage)),
		strconv.Itoa(int(c.TotalPerNodeMemoryOverage)),
		strconv.Itoa(int(c.TotalClusterCPUOverage)),
		strconv.Itoa(int(c.TotalClusterMemoryOverate)),
	}
}

func (na NodeAllocated) GetMemoryAllocatableReservation() resourceapi.Quantity {
	brackets := []reservedBracket{
		{
			threshold: resourceapi.MustParse("400Gi"),
			reserved:  resourceapi.MustParse("14Gi"),
		},
		{
			threshold: resourceapi.MustParse("250Gi"),
			reserved:  resourceapi.MustParse("11Gi"),
		},
		{
			threshold: resourceapi.MustParse("120Gi"),
			reserved:  resourceapi.MustParse("8Gi"),
		},
		{
			threshold: resourceapi.MustParse("63Gi"),
			reserved:  resourceapi.MustParse("6Gi"),
		},
		{
			threshold: resourceapi.MustParse("31Gi"),
			reserved:  resourceapi.MustParse("3.5Gi"),
		},
		{
			threshold: resourceapi.MustParse("15Gi"),
			reserved:  resourceapi.MustParse("2.75Gi"),
		},
		{
			threshold: resourceapi.MustParse("7Gi"),
			reserved:  resourceapi.MustParse("2Gi"),
		},
		{
			threshold: resourceapi.MustParse("0"),
			reserved:  resourceapi.MustParse("0"),
		},
	}
	for _, bracket := range brackets {
		if na.MemoryAllocatable.Cmp(bracket.threshold) > 0 {
			if bracket.reserved.IsZero() {
				evictionThreshold := resourceapi.MustParse("250Mi")
				return *resourceapi.NewQuantity(na.MemoryAllocatable.Value()/4+evictionThreshold.Value(), resourceapi.DecimalSI)
			}
			return bracket.reserved
		}
	}
	return *resourceapi.NewQuantity(0, resourceapi.DecimalSI)
}

func (na NodeAllocated) GetCPUAllocatableReservation() resourceapi.Quantity {
	brackets := []reservedBracket{
		{
			threshold: resourceapi.MustParse("48000m"),
			reserved:  resourceapi.MustParse("300m"),
		},
		{
			threshold: resourceapi.MustParse("24000m"),
			reserved:  resourceapi.MustParse("150m"),
		},
		{
			threshold: resourceapi.MustParse("12000m"),
			reserved:  resourceapi.MustParse("120m"),
		},
		{
			threshold: resourceapi.MustParse("6000m"),
			reserved:  resourceapi.MustParse("90m"),
		},
		{
			threshold: resourceapi.MustParse("3000m"),
			reserved:  resourceapi.MustParse("80m"),
		},
		{
			threshold: resourceapi.MustParse("1500m"),
			reserved:  resourceapi.MustParse("70m"),
		},
		{
			threshold: *resourceapi.NewQuantity(0, resourceapi.DecimalSI),
			reserved:  resourceapi.MustParse("60m"),
		},
	}
	for _, bracket := range brackets {
		if na.CPUAllocatable.Cmp(bracket.threshold) > 0 {
			return bracket.reserved
		}
	}
	return *resourceapi.NewQuantity(0, resourceapi.DecimalSI)
}

func (na *NodeAllocated) String() string {
	return fmt.Sprintf(allocatableTemplate, na.NodeName, na.MemoryRequests.String(), na.MemoryAllocatable.String(), na.GetMemoryPercent(), na.CPURequests.String(), na.CPUAllocatable.String(), na.GetCPUPercent())
}

func (na *NodeAllocated) GetMemoryPercent() int64 {
	return 100.0 * na.MemoryRequests.Value() / na.MemoryAllocatable.Value()
}

func (na *NodeAllocated) GetCPUPercent() int64 {
	return 100.0 * na.CPURequests.MilliValue() / na.CPUAllocatable.MilliValue()
}

func ParseClusterInfo(input string) (*ClusterInfo, error) {
	return nil, fmt.Errorf("Not implemented")
}

func GetClusterInfo(pods []v1.Pod, nodes []v1.Node) *ClusterInfo {
	millicores := int64(0)
	for _, node := range nodes {
		cpu := node.Status.Capacity[v1.ResourceCPU]
		millicores += cpu.MilliValue()
	}
	numPods := 0
	for _, pod := range pods {
		if pod.Status.Phase == v1.PodRunning {
			numPods++
		}
	}
	version := ""
	if len(nodes) > 0 {
		version = nodes[0].Status.NodeInfo.KubeletVersion
	}

	return &ClusterInfo{
		pods:        numPods,
		nodes:       len(nodes),
		cores:       int(millicores / 1000.0),
		nodeVersion: version,
	}
}

func (c *ClusterInfo) String() string {
	return fmt.Sprintf(clusterInfoTemplate, c.pods, c.nodes, c.cores, c.nodeVersion)
}

func ParseDisruptiveEventList(input string) (DisruptiveEventList, error) {
	return nil, fmt.Errorf("Not implemented")
}

func GetDisruptiveEventList(events []v1.Event) DisruptiveEventList {
	disruptiveReasons := []string{"Evicted", "OOMKilling", "SystemOOM"}
	filteredEvents := []v1.Event{}
	for _, event := range events {
		for _, reason := range disruptiveReasons {
			if event.Reason == reason {
				filteredEvents = append(filteredEvents, event)
			}
		}
	}
	return filteredEvents
}

func (d DisruptiveEventList) String() string {
	eventString := ""
	for _, event := range d {
		eventString += fmt.Sprintf(eventTemplate, event.Reason, event.Message, event.Count)
		eventString += ";"
	}
	return strings.TrimSuffix(eventString, ",")
}
