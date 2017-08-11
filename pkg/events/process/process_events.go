package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/dashpole/allocatable/pkg/common"
	"github.com/dashpole/allocatable/pkg/events/types"
)

var path = flag.String("path", "foreachmaster.log", "path to your log file")
var outputFile = flag.String("output", "_output/eventStats.csv", "path to output file")

func main() {
	flag.Parse()
	file, err := os.Open(*path)
	if err != nil {
		fmt.Printf("Error opening file: %v", err)
	}
	defer file.Close()

	data := [][]string{}
	r := bufio.NewReaderSize(file, 512*1024)
	line, bufferToSmall, err := r.ReadLine()
	for err == nil && !bufferToSmall {
		_, clusterLines, parseErr := common.ParseForeachMasterLine(line)
		if parseErr == nil {
			/*
				Lines are as follows:
				0: "starting shell script"
				1: "Getting Events"
				2...n-4: Events
				n-3: "Getting ClusterInfo"
				n-2: ClusterInfo
				n-1:
			*/
			clusterData := []string{}
			clusterInfo, parseErr := types.ParseClusterInfo(clusterLines[len(clusterLines)-2])
			if parseErr == nil {
				clusterData = append(clusterData, clusterInfo.ToSlice()...)
				eventList := types.ParseDisruptiveEventList(strings.Join(clusterLines[2:len(clusterLines)-3], ""))
				clusterData = append(clusterData, eventList.ToSlice()...)
				data = append(data, clusterData)
			}
		}
		line, bufferToSmall, err = r.ReadLine()
	}
	if bufferToSmall {
		fmt.Println("buffer size to small")
		return
	}
	if err != io.EOF {
		fmt.Println(err)
		return
	}

	err = common.ToCSV(*outputFile, data)
	if err != nil {
		fmt.Printf("Error writing output to csv: %v\n", err)
	}
}
