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
)

var path = flag.String("path", "foreachmaster.log", "path to your log file")

const clusterExpr = `^\{.*\} output: \"(.*)\"$`
const nodeExpr = `^NodeName: (.*), Memory: (.*) / (.*) = .*, CPU: (.*) / (.*) = .*$`

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

type outputType struct {
	sliceFunc toSliceFunc
	fileName  string
	data      [][]string
}

func main() {
	flag.Parse()
	file, err := os.Open(*path)
	if err != nil {
		fmt.Printf("Error opening file: %v", err)
	}
	defer file.Close()

	allClusterStats := []ClusterStats{}
	allNodeStats := []NodeAllocated{}
	r := bufio.NewReaderSize(file, 512*1024)
	line, isPrefix, err := r.ReadLine()
	for err == nil && !isPrefix {
		clusterAllocated := parseCluster(line)
		if len(clusterAllocated) > 0 {
			allNodeStats = append(allNodeStats, clusterAllocated...)
			allClusterStats = append(allClusterStats, clusterAllocated.getStats())
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
		data = append(data, cluster.toSlice())
	}
	err = toCSV("specificClusterStats.csv", data)
	if err != nil {
		fmt.Printf("Error writing output to csv: %v\n", err)
	}

	// err = outputClusterToCsv(allClusterStats, allNodeStats)
	// if err != nil {
	// 	fmt.Println(err)
	// }
}

func outputClusterToCsv(clusterStatsList []ClusterStats, nodeStatsList []NodeAllocated) error {
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

func parseCluster(input []byte) ClusterAllocated {
	clusterAllocated := []NodeAllocated{}
	re := regexp.MustCompile(clusterExpr)
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

func parseNode(inputNode string) *NodeAllocated {
	re := regexp.MustCompile(nodeExpr)
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

func (c ClusterAllocated) getStats() ClusterStats {
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
		cpuReserved := na.getCPUAllocatableReservation()
		memoryReserved := na.getMemoryAllocatableReservation()
		totalCPUReserved.Add(cpuReserved)
		totalMemoryReserved.Add(memoryReserved)

		perNodeCPUOverage := na.CPUAllocatable.MilliValue()
		perNodeCPUOverage -= na.CPURequests.MilliValue()
		perNodeCPUOverage -= cpuReserved.MilliValue()
		if perNodeCPUOverage > 0 {
			totalPerNodeCPUOverage += perNodeCPUOverage
		}

		perNodeMemoryOverage := na.MemoryAllocatable.Value()
		perNodeMemoryOverage -= na.MemoryRequests.Value()
		perNodeMemoryOverage -= memoryReserved.Value()
		if perNodeMemoryOverage > 0 {
			totalPerNodeMemoryOverage += perNodeMemoryOverage
		}
	}
	clusterCPUOverage := totalCPUAllocatable.Copy()
	clusterCPUOverage.Sub(*totalCPURequests)
	clusterCPUOverage.Sub(*totalCPUReserved)
	clusterMemoryOverage := totalMemoryAllocatable.Copy()
	clusterMemoryOverage.Sub(*totalMemoryRequests)
	clusterMemoryOverage.Sub(*totalMemoryReserved)
	return ClusterStats{
		NumNodes:                  len(c),
		ClusterCPU:                totalCPUAllocatable.MilliValue(),
		ClusterMemory:             totalMemoryAllocatable.Value(),
		MaxMemPercentage:          maxMemoryPerc,
		MaxCPUPercentage:          maxCPUPerc,
		MeanMemPercentage:         100 * totalMemoryRequests.Value() / totalMemoryAllocatable.Value(),
		MeanCPUPercentage:         100 * totalCPURequests.MilliValue() / totalCPUAllocatable.MilliValue(),
		TotalPerNodeCPUOverage:    totalPerNodeCPUOverage,
		TotalClusterCPUOverage:    clusterCPUOverage.MilliValue(),
		TotalPerNodeMemoryOverage: totalPerNodeMemoryOverage,
		TotalClusterMemoryOverate: clusterMemoryOverage.Value(),
	}
}

func getAggregateStats(clusterStatsList []ClusterStats, nodeStatsList []NodeAllocated, memThreshold int64, cpuThreshold int64) AggregateStats {
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
	return AggregateStats{
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

type toSliceFunc func(a AggregateStats) []string

func toClusterSlice(a AggregateStats) []string {
	return []string{strconv.Itoa(a.CPUThreshold), strconv.Itoa(a.MemThreshold), strconv.Itoa(a.ClustersAffected), strconv.Itoa(a.ClustersUnknown), strconv.Itoa(a.ClustersUnaffected)}
}

func toClusterNodeSlice(a AggregateStats) []string {
	return []string{strconv.Itoa(a.CPUThreshold), strconv.Itoa(a.MemThreshold), strconv.Itoa(a.NodesInAffectedClusters), strconv.Itoa(a.NodesInUnknownClusters), strconv.Itoa(a.NodesInUnaffectedClusters)}
}

func toNodeSlice(a AggregateStats) []string {
	return []string{strconv.Itoa(a.CPUThreshold), strconv.Itoa(a.MemThreshold), strconv.Itoa(a.NodesAffected), strconv.Itoa(a.NodesUnaffected)}
}

func toClusterSizeSlice(a AggregateStats) []string {
	return []string{strconv.Itoa(a.CPUThreshold), strconv.Itoa(a.MemThreshold), strconv.Itoa(a.NodesInAffectedClusters / a.ClustersAffected), strconv.Itoa(a.NodesInUnknownClusters / a.ClustersUnknown)}
}

func (c ClusterStats) toSlice() []string {
	return []string{
		strconv.Itoa(int(c.NumNodes)),
		strconv.Itoa(int(c.ClusterCPU)),
		strconv.Itoa(int(c.ClusterMemory)),
		strconv.Itoa(int(c.TotalPerNodeCPUOverage)),
		strconv.Itoa(int(c.TotalPerNodeMemoryOverage)),
		strconv.Itoa(int(c.TotalClusterCPUOverage)),
		strconv.Itoa(int(c.TotalClusterMemoryOverate)),
	}
}

type reservedBracket struct {
	threshold resourceapi.Quantity
	reserved  resourceapi.Quantity
}

func (na NodeAllocated) getMemoryAllocatableReservation() resourceapi.Quantity {
	brackets := []reservedBracket{
		{
			threshold: resourceapi.MustParse("7Gi"),
			reserved:  resourceapi.MustParse("1Gi"),
		},
		{
			threshold: resourceapi.MustParse("3Gi"),
			reserved:  resourceapi.MustParse("750Mi"),
		},
		{
			threshold: *resourceapi.NewQuantity(0, resourceapi.DecimalSI),
			reserved:  resourceapi.MustParse("500Mi"),
		},
	}
	for _, bracket := range brackets {
		if na.MemoryAllocatable.Cmp(bracket.threshold) > 0 {
			return bracket.reserved
		}
	}
	return *resourceapi.NewQuantity(0, resourceapi.DecimalSI)
}

func (na NodeAllocated) getCPUAllocatableReservation() resourceapi.Quantity {
	brackets := []reservedBracket{
		{
			threshold: resourceapi.MustParse("4000m"),
			reserved:  resourceapi.MustParse("300m"),
		},
		{
			threshold: resourceapi.MustParse("1000m"),
			reserved:  resourceapi.MustParse("200m"),
		},
		{
			threshold: *resourceapi.NewQuantity(0, resourceapi.DecimalSI),
			reserved:  resourceapi.MustParse("100m"),
		},
	}
	for _, bracket := range brackets {
		if na.CPUAllocatable.Cmp(bracket.threshold) > 0 {
			return bracket.reserved
		}
	}
	return *resourceapi.NewQuantity(0, resourceapi.DecimalSI)
}
