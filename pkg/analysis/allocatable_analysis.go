package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	resourceapi "k8s.io/apimachinery/pkg/api/resource"

	"github.com/dashpole/allocatable/pkg/types"
)

var path = flag.String("path", "foreachmaster.log", "path to your log file")

func main() {
	flag.Parse()
	file, err := os.Open(*path)
	if err != nil {
		fmt.Printf("Error opening file: %v", err)
	}
	defer file.Close()

	allClusterStats := []types.ClusterStats{}
	allNodeStats := []types.NodeAllocated{}
	r := bufio.NewReaderSize(file, 512*1024)
	line, isPrefix, err := r.ReadLine()
	for err == nil && !isPrefix {
		clusterAllocated := parseCluster(line)
		if len(clusterAllocated) > 0 {
			allNodeStats = append(allNodeStats, clusterAllocated...)
			allClusterStats = append(allClusterStats, getClusterStats(clusterAllocated))
		}
		line, isPrefix, err = r.ReadLine()
	}
	if isPrefix {
		fmt.Println("buffer size to small")
		return
	}
	if err != io.EOF {
		fmt.Println(err)
		return
	}

	data := [][]string{}
	for _, cluster := range allClusterStats {
		data = append(data, cluster.ToSlice())
	}
	err = toCSV("../../_output/specificClusterStats.csv", data)
	if err != nil {
		fmt.Printf("Error writing output to csv: %v\n", err)
	}
}

func outputClusterToCsv(clusterStatsList []types.ClusterStats, nodeStatsList []types.NodeAllocated) error {
	outputs := []outputType{
		{
			sliceFunc: toClusterSlice,
			fileName:  "clusterStats.csv",
			data:      [][]string{},
		},
		{
			sliceFunc: toClusterNodeSlice,
			fileName:  "clusterNodeStats.csv",
			data:      [][]string{},
		},
		{
			sliceFunc: toNodeSlice,
			fileName:  "nodeStats.csv",
			data:      [][]string{},
		},
		{
			sliceFunc: toClusterSizeSlice,
			fileName:  "clusterSizeStats.csv",
			data:      [][]string{},
		},
	}
	for i := 0; i < 100; i++ {
		aggregateStats := getAggregateStats(clusterStatsList, nodeStatsList, int64(0), int64(i))
		for i, output := range outputs {
			outputs[i].data = append(output.data, output.sliceFunc(aggregateStats))
		}
	}
	for _, output := range outputs {
		err := toCSV(output.fileName, output.data)
		if err != nil {
			return fmt.Errorf("Error writing output to csv: %v\n", err)
		}
	}
	return nil
}

func toCSV(filename string, data [][]string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	err = writer.WriteAll(data)
	if err != nil {
		return err
	}
	return nil
}

func parseCluster(input []byte) types.ClusterAllocated {
	clusterAllocated := []types.NodeAllocated{}
	re := regexp.MustCompile(types.ClusterExpr)
	if re.Match(input) {
		// get the portion captured by parenthesis in the expr
		match := re.FindSubmatch(input)[1]
		nodes := strings.Split(string(match), "\\n")
		for _, node := range nodes {
			nodeAllocated := parseNode(node)
			if nodeAllocated != nil {
				clusterAllocated = append(clusterAllocated, *nodeAllocated)
			}
		}
	}
	return clusterAllocated
}

func parseNode(inputNode string) *types.NodeAllocated {
	re := regexp.MustCompile(types.NodeExpr)
	if re.MatchString(inputNode) {
		// get the portion captured by parenthesis in the expr
		matches := re.FindStringSubmatch(inputNode)
		return &types.NodeAllocated{
			NodeName:          matches[1],
			MemoryAllocatable: resourceapi.MustParse(matches[3]),
			CPUAllocatable:    resourceapi.MustParse(matches[5]),
			MemoryRequests:    resourceapi.MustParse(matches[2]),
			CPURequests:       resourceapi.MustParse(matches[4]),
		}
	}
	return nil
}

func getClusterStats(c types.ClusterAllocated) types.ClusterStats {
	maxCPUPerc := int64(0)
	maxMemoryPerc := int64(0)
	totalPerNodeCPUOverage := int64(0)
	totalPerNodeMemoryOverage := int64(0)
	totalCPURequests := resourceapi.NewQuantity(0, resourceapi.DecimalSI)
	totalCPUAllocatable := resourceapi.NewQuantity(0, resourceapi.DecimalSI)
	totalMemoryRequests := resourceapi.NewQuantity(0, resourceapi.DecimalSI)
	totalMemoryAllocatable := resourceapi.NewQuantity(0, resourceapi.DecimalSI)
	totalCPUReserved := resourceapi.NewQuantity(0, resourceapi.DecimalSI)
	totalMemoryReserved := resourceapi.NewQuantity(0, resourceapi.DecimalSI)
	for _, na := range c {
		memoryPercent := na.GetMemoryPercent()
		if memoryPercent >= maxMemoryPerc {
			maxMemoryPerc = memoryPercent
		}
		cpuPercent := na.GetCPUPercent()
		if cpuPercent >= maxCPUPerc {
			maxCPUPerc = cpuPercent
		}
		totalCPURequests.Add(na.CPURequests)
		totalCPUAllocatable.Add(na.CPUAllocatable)
		totalMemoryRequests.Add(na.MemoryRequests)
		totalMemoryAllocatable.Add(na.MemoryAllocatable)
		cpuReserved := na.GetCPUAllocatableReservation()
		memoryReserved := na.GetMemoryAllocatableReservation()
		totalCPUReserved.Add(cpuReserved)
		totalMemoryReserved.Add(memoryReserved)

		perNodeCPUOverage := na.CPURequests.MilliValue() + cpuReserved.MilliValue() - na.CPUAllocatable.MilliValue()
		if perNodeCPUOverage > 0 {
			totalPerNodeCPUOverage += perNodeCPUOverage
		}

		perNodeMemoryOverage := na.MemoryRequests.Value() + memoryReserved.Value() - na.MemoryAllocatable.Value()
		if perNodeMemoryOverage > 0 {
			totalPerNodeMemoryOverage += perNodeMemoryOverage
		}
	}
	clusterCPUOverage := totalCPURequests.MilliValue() + totalCPUReserved.MilliValue() - totalCPUAllocatable.MilliValue()
	if clusterCPUOverage < 0 {
		clusterCPUOverage = 0
	}
	clusterMemoryOverage := totalMemoryRequests.Value() + totalMemoryReserved.Value() - totalMemoryAllocatable.Value()
	if clusterMemoryOverage < 0 {
		clusterMemoryOverage = 0
	}
	return types.ClusterStats{
		NumNodes:                  len(c),
		ClusterCPU:                totalCPUAllocatable.MilliValue(),
		ClusterMemory:             totalMemoryAllocatable.Value(),
		ClusterCPUReserved:        totalCPUReserved.MilliValue(),
		ClusterMemoryReserved:     totalMemoryReserved.Value(),
		MaxMemPercentage:          maxMemoryPerc,
		MaxCPUPercentage:          maxCPUPerc,
		MeanMemPercentage:         100 * totalMemoryRequests.Value() / totalMemoryAllocatable.Value(),
		MeanCPUPercentage:         100 * totalCPURequests.MilliValue() / totalCPUAllocatable.MilliValue(),
		TotalPerNodeCPUOverage:    totalPerNodeCPUOverage,
		TotalClusterCPUOverage:    clusterCPUOverage,
		TotalPerNodeMemoryOverage: totalPerNodeMemoryOverage,
		TotalClusterMemoryOverate: clusterMemoryOverage,
	}
}

func getAggregateStats(clusterStatsList []types.ClusterStats, nodeStatsList []types.NodeAllocated, memThreshold int64, cpuThreshold int64) types.AggregateStats {
	clustersAffected := 0
	nodesInAffectedClusters := 0
	clustersUnknown := 0
	nodesInUnknownClusters := 0
	clustersUnaffected := 0
	nodesInUnaffectedClusters := 0
	nodesAffected := 0
	nodesUnaffected := 0
	for _, cs := range clusterStatsList {
		if cs.MeanMemPercentage > memThreshold || cs.MeanCPUPercentage > cpuThreshold {
			clustersAffected++
			nodesInAffectedClusters += cs.NumNodes
		} else if cs.MaxMemPercentage < memThreshold && cs.MaxCPUPercentage < cpuThreshold {
			clustersUnaffected++
			nodesInUnaffectedClusters += cs.NumNodes
		} else {
			clustersUnknown++
			nodesInUnknownClusters += cs.NumNodes
		}
	}
	for _, na := range nodeStatsList {
		if na.GetMemoryPercent() < memThreshold && na.GetCPUPercent() < cpuThreshold {
			nodesUnaffected++
		} else {
			nodesAffected++
		}
	}
	return types.AggregateStats{
		CPUThreshold:              int(cpuThreshold),
		MemThreshold:              int(memThreshold),
		ClustersAffected:          clustersAffected,
		NodesInAffectedClusters:   nodesInAffectedClusters,
		ClustersUnknown:           clustersUnknown,
		NodesInUnknownClusters:    nodesInUnknownClusters,
		ClustersUnaffected:        clustersUnaffected,
		NodesInUnaffectedClusters: nodesInUnaffectedClusters,
		NodesAffected:             nodesAffected,
		NodesUnaffected:           nodesUnaffected,
	}
}

type toSliceFunc func(a types.AggregateStats) []string

func toClusterSlice(a types.AggregateStats) []string {
	return []string{strconv.Itoa(a.CPUThreshold), strconv.Itoa(a.MemThreshold), strconv.Itoa(a.ClustersAffected), strconv.Itoa(a.ClustersUnknown), strconv.Itoa(a.ClustersUnaffected)}
}

func toClusterNodeSlice(a types.AggregateStats) []string {
	return []string{strconv.Itoa(a.CPUThreshold), strconv.Itoa(a.MemThreshold), strconv.Itoa(a.NodesInAffectedClusters), strconv.Itoa(a.NodesInUnknownClusters), strconv.Itoa(a.NodesInUnaffectedClusters)}
}

func toNodeSlice(a types.AggregateStats) []string {
	return []string{strconv.Itoa(a.CPUThreshold), strconv.Itoa(a.MemThreshold), strconv.Itoa(a.NodesAffected), strconv.Itoa(a.NodesUnaffected)}
}

func toClusterSizeSlice(a types.AggregateStats) []string {
	return []string{strconv.Itoa(a.CPUThreshold), strconv.Itoa(a.MemThreshold), strconv.Itoa(a.NodesInAffectedClusters / a.ClustersAffected), strconv.Itoa(a.NodesInUnknownClusters / a.ClustersUnknown)}
}

type outputType struct {
	sliceFunc toSliceFunc
	fileName  string
	data      [][]string
}
