package types

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"k8s.io/api/core/v1"
)

const (
	clusterInfoExpr     = `^Pods: (.*), Nodes: (.*), Cores: (.*), NodeVersion: (.*)$`
	clusterInfoTemplate = "Pods: %d, Nodes: %d, Cores: %d, NodeVersion: %s"
	eventExpr           = `^Reason: (.*), Message: (.*), Count: (.*)$`
	eventTemplate       = "Reason: %v, Message: %v, Count: %v"
)

type ClusterInfo struct {
	pods        int
	nodes       int
	cores       int
	nodeVersion string
}

type DisruptiveEventList []v1.Event

func ParseClusterInfo(input string) (*ClusterInfo, error) {
	re := regexp.MustCompile(clusterInfoExpr)
	if re.MatchString(input) {
		submatches := re.FindStringSubmatch(input)
		pods, err := strconv.Atoi(submatches[1])
		if err != nil {
			return nil, err
		}
		nodes, err := strconv.Atoi(submatches[2])
		if err != nil {
			return nil, err
		}
		cores, err := strconv.Atoi(submatches[3])
		if err != nil {
			return nil, err
		}
		return &ClusterInfo{
			pods:        pods,
			nodes:       nodes,
			cores:       cores,
			nodeVersion: submatches[4],
		}, nil
	}
	return nil, fmt.Errorf("Unable to parse line, clusterInfo: %s did not match expr: %s", string(input), clusterInfoExpr)
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

func (c *ClusterInfo) ToSlice() []string {
	return []string{strconv.Itoa(c.pods), strconv.Itoa(c.nodes), strconv.Itoa(c.cores), c.nodeVersion}
}

func ParseDisruptiveEventList(input string) DisruptiveEventList {
	events := strings.Split(input, ";")
	eventList := []v1.Event{}
	for _, eventString := range events {
		event, err := ParseEvent(eventString)
		if err == nil {
			eventList = append(eventList, *event)
		}
	}
	return AggregateEvents(eventList)
}

func ParseEvent(input string) (*v1.Event, error) {
	re := regexp.MustCompile(eventExpr)
	if re.MatchString(input) {
		submatches := re.FindStringSubmatch(input)
		count, err := strconv.ParseInt(submatches[3], 10, 32)
		if err != nil {
			return nil, err
		}
		return &v1.Event{
			Reason:  submatches[1],
			Message: submatches[2],
			Count:   int32(count),
		}, nil
	}
	return nil, fmt.Errorf("Unable to parse event, input: %s did not match expr: %s", string(input), eventExpr)
}

func AggregateEvents(inputEvents []v1.Event) []v1.Event {
	outputEvents := []v1.Event{}
	eventMap := make(map[string]int32)
	for _, event := range inputEvents {
		eventMap[event.Reason] += event.Count
	}
	for k, v := range eventMap {
		outputEvents = append(outputEvents, v1.Event{
			Reason: k,
			Count:  v,
		})
	}
	return outputEvents
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
	return strings.TrimSuffix(eventString, ";")
}

func (d DisruptiveEventList) ToSlice() []string {
	eventSlice := []string{}
	for _, event := range d {
		eventSlice = append(eventSlice, event.Reason, strconv.FormatInt(int64(event.Count), 10))
	}
	return eventSlice
}
