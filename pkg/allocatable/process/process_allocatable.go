package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"

	resourceapi "k8s.io/apimachinery/pkg/api/resource"

	"github.com/dashpole/allocatable/pkg/allocatable/types"
	"github.com/dashpole/allocatable/pkg/common"
)

var path = flag.String("path", "foreachmaster.log", "path to your log file")
var outputFile = flag.String("output", "_output/specificClusterStats.csv", "path to output file")

func main() {
	flag.Parse()
	file, err := os.Open(*path)
	if err != nil {
		fmt.Printf("Error opening file: %v", err)
	}
	defer file.Close()

	allClusterStats := []types.ClusterStats{}
	r := bufio.NewReaderSize(file, 512*1024)
	line, isPrefix, err := r.ReadLine()
	for err == nil && !isPrefix {
		clusterAllocated, id := types.ParseClusterAllocated(line)
		if len(clusterAllocated) > 0 {
			allClusterStats = append(allClusterStats, getClusterStats(clusterAllocated, id))
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

	data := [][]string{types.GetClusterStatsHeader()}
	for _, cluster := range allClusterStats {
		if cluster.IsAffected() {
			data = append(data, cluster.ToSlice())
		}
	}
	err = common.ToCSV(*outputFile, data)
	if err != nil {
		fmt.Printf("Error writing output to csv: %v\n", err)
	}
}

func getClusterStats(c types.ClusterAllocated, id string) types.ClusterStats {
	totalPerNodeCPUOverage := int64(0)
	totalPerNodeMemoryOverage := int64(0)
	totalCPURequests := resourceapi.NewQuantity(0, resourceapi.DecimalSI)
	totalCPUAllocatable := resourceapi.NewQuantity(0, resourceapi.DecimalSI)
	totalMemoryRequests := resourceapi.NewQuantity(0, resourceapi.DecimalSI)
	totalMemoryAllocatable := resourceapi.NewQuantity(0, resourceapi.DecimalSI)
	totalCPUReserved := resourceapi.NewQuantity(0, resourceapi.DecimalSI)
	totalMemoryReserved := resourceapi.NewQuantity(0, resourceapi.DecimalSI)
	for _, na := range c {
		totalCPURequests.Add(na.CPURequests)
		totalCPUAllocatable.Add(na.CPUAllocatable)
		totalMemoryRequests.Add(na.MemoryRequests)
		totalMemoryAllocatable.Add(na.MemoryAllocatable)
		cpuReserved := getCPUReservation(na.CPUAllocatable.MilliValue())
		memoryReserved := getMemoryReservation(na.MemoryAllocatable.Value())
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
		TotalPerNodeCPUOverage:    totalPerNodeCPUOverage,
		TotalClusterCPUOverage:    clusterCPUOverage,
		TotalPerNodeMemoryOverage: totalPerNodeMemoryOverage,
		TotalClusterMemoryOverage: clusterMemoryOverage,
		Identifier:                id,
	}
}
