package types

import (
	"fmt"
	"regexp"
	"strconv"

	resourceapi "k8s.io/apimachinery/pkg/api/resource"

	"github.com/dashpole/allocatable/pkg/common"
)

type ClusterStats struct {
	NumNodes                  int
	ClusterCPU                int64
	ClusterMemory             int64
	ClusterCPUReserved        int64
	ClusterMemoryReserved     int64
	TotalPerNodeCPUOverage    int64
	TotalClusterCPUOverage    int64
	TotalPerNodeMemoryOverage int64
	TotalClusterMemoryOverage int64
	Identifier                string
}

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
		strconv.Itoa(int(c.TotalClusterMemoryOverage)),
		c.Identifier,
	}
}

func GetClusterStatsHeader() []string {
	return []string{
		"Nodes",
		"CPU Capacity",
		"Memory Capacity",
		"CPU Reserved",
		"Memory Reserved",
		"Node CPU Overage",
		"Node Memory Overage",
		"Cluster CPU Overage",
		"Cluster Memory Overage",
		"Identifier",
	}
}

func (c ClusterStats) IsAffected() bool {
	if c.TotalClusterCPUOverage > 0 && c.TotalClusterCPUOverage < c.ClusterCPUReserved {
		// affected by CPU
		return true
	} else if c.TotalClusterMemoryOverage > 0 && c.TotalClusterMemoryOverage < c.ClusterMemoryReserved {
		return true
	}
	return false
}

type ClusterAllocated []NodeAllocated

func ParseClusterAllocated(input []byte) (ClusterAllocated, string) {
	clusterAllocated := []NodeAllocated{}
	id, lines, err := common.ParseForeachMasterLine(input)
	if err == nil {
		for _, node := range lines {
			nodeAllocated := parseNodeAllocated(node)
			if nodeAllocated != nil {
				clusterAllocated = append(clusterAllocated, *nodeAllocated)
			}
		}
	}
	return clusterAllocated, id
}

type NodeAllocated struct {
	NodeName          string
	MemoryAllocatable resourceapi.Quantity
	CPUAllocatable    resourceapi.Quantity
	MemoryRequests    resourceapi.Quantity
	CPURequests       resourceapi.Quantity
}

const NodeExpr = `^NodeName: (.*), Memory: (.*) / (.*) = .*, CPU: (.*) / (.*) = .*$`

func parseNodeAllocated(inputNode string) *NodeAllocated {
	re := regexp.MustCompile(NodeExpr)
	if re.MatchString(inputNode) {
		// get the portion captured by parenthesis in the expr
		matches := re.FindStringSubmatch(inputNode)
		return &NodeAllocated{
			NodeName:          matches[1],
			MemoryAllocatable: resourceapi.MustParse(matches[3]),
			CPUAllocatable:    resourceapi.MustParse(matches[5]),
			MemoryRequests:    resourceapi.MustParse(matches[2]),
			CPURequests:       resourceapi.MustParse(matches[4]),
		}
	}
	return nil
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
