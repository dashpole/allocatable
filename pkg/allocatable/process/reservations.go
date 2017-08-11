package main

import (
	"fmt"

	resourceapi "k8s.io/apimachinery/pkg/api/resource"
)

const (
	mbPerGB           = 1024
	millicoresPerCore = 1000
)

func getMemoryReservation(memoryAllocatableBytes int64) resourceapi.Quantity {
	return resourceapi.MustParse(fmt.Sprintf("%dMi", memoryReservedMB(memoryAllocatableBytes/mbPerGB/mbPerGB)))
}

func getCPUReservation(cpuAllocatableMillicores int64) resourceapi.Quantity {
	return resourceapi.MustParse(fmt.Sprintf("%dm", cpuReservedMillicores(cpuAllocatableMillicores)))
}

type allocatableBracket struct {
	threshold            int64
	marginalReservedRate float64
}

func memoryReservedMB(memoryCapacityMB int64) int64 {
	if memoryCapacityMB <= 1*mbPerGB {
		// do not set any memory reserved for nodes with less than 1 Gb of capacity
		return 0
	}
	return calculateReserved(memoryCapacityMB, []allocatableBracket{
		{
			threshold:            0,
			marginalReservedRate: 0.25,
		},
		{
			threshold:            4 * mbPerGB,
			marginalReservedRate: 0.2,
		},
		{
			threshold:            8 * mbPerGB,
			marginalReservedRate: 0.1,
		},
		{
			threshold:            16 * mbPerGB,
			marginalReservedRate: 0.06,
		},
		{
			threshold:            128 * mbPerGB,
			marginalReservedRate: 0.02,
		},
	})
}

func cpuReservedMillicores(cpuCapacityMillicores int64) int64 {
	return calculateReserved(cpuCapacityMillicores, []allocatableBracket{
		{
			threshold:            0,
			marginalReservedRate: 0.06,
		},
		{
			threshold:            1 * millicoresPerCore,
			marginalReservedRate: 0.01,
		},
		{
			threshold:            2 * millicoresPerCore,
			marginalReservedRate: 0.005,
		},
		{
			threshold:            4 * millicoresPerCore,
			marginalReservedRate: 0.0025,
		},
	})
}

// calculateReserved calculates reserved using capacity and a series of
// brackets as follows:  the marginalReservedRate applies to all capacity
// greater than the bracket, but less than the next bracket.  For example, if
// the first bracket is threshold: 0, rate:0.1, and the second bracket has
// threshold: 100, rate: 0.4, a capacity of 100 results in a reserved of
// 100*0.1 = 10, but a capacity of 200 results in a reserved of
// 10 + (200-100)*.4 = 50.  Using brackets with marginal rates ensures that as
// capacity increases, reserved always increases, and never decreases.
func calculateReserved(capacity int64, brackets []allocatableBracket) int64 {
	var reserved float64
	for i, bracket := range brackets {
		c := capacity
		if i < len(brackets)-1 && brackets[i+1].threshold < capacity {
			c = brackets[i+1].threshold
		}
		additionalReserved := float64(c-bracket.threshold) * bracket.marginalReservedRate
		if additionalReserved > 0 {
			reserved += additionalReserved
		}
	}
	return int64(reserved)
}
