package types

import (
	"k8s.io/api/core/v1"
	resourceapi "k8s.io/apimachinery/pkg/api/resource"
)

type AggregateStats struct {
	CPUThreshold              int
	MemThreshold              int
	ClustersAffected          int
	ClustersUnknown           int
	ClustersUnaffected        int
	NodesInAffectedClusters   int
	NodesInUnknownClusters    int
	NodesInUnaffectedClusters int
	NodesAffected             int
	NodesUnaffected           int
}

type ClusterStats struct {
	NumNodes                  int
	ClusterCPU                int64
	ClusterMemory             int64
	ClusterCPUReserved        int64
	ClusterMemoryReserved     int64
	MaxMemPercentage          int64
	MaxCPUPercentage          int64
	MeanMemPercentage         int64
	MeanCPUPercentage         int64
	TotalPerNodeCPUOverage    int64
	TotalClusterCPUOverage    int64
	TotalPerNodeMemoryOverage int64
	TotalClusterMemoryOverate int64
}

type ClusterAllocated []NodeAllocated

type NodeAllocated struct {
	NodeName          string
	MemoryAllocatable resourceapi.Quantity
	CPUAllocatable    resourceapi.Quantity
	MemoryRequests    resourceapi.Quantity
	CPURequests       resourceapi.Quantity
}

const ClusterExpr = `^\{.*\} output: \"(.*)\"$`
const NodeExpr = `^NodeName: (.*), Memory: (.*) / (.*) = .*, CPU: (.*) / (.*) = .*$`

type reservedBracket struct {
	threshold resourceapi.Quantity
	reserved  resourceapi.Quantity
}

const allocatableTemplate = "NodeName: %s, Memory: %s / %s = %v%%, CPU: %s / %s = %v%%"

const clusterInfoTemplate = "Pods: %d, Nodes: %d, Cores: %d, NodeVersion: %s"

type ClusterInfo struct {
	pods        int
	nodes       int
	cores       int
	nodeVersion string
}

const eventTemplate = "Reason: %v, Message: %v, Count: %v"

type DisruptiveEventList []v1.Event
