#!/usr/bin/env bash

# Copyright 2017 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

PODS="kubectl get pods --all-namespaces=true -o json"
POD_REQ=$(${PODS} | jq '[.items[] | {node_name: .spec.nodeName, cpu_req: [.spec.containers[].resources.requests.cpu], memory_req: [.spec.containers[].resources.requests.memory], "pods": 1}]')
MERGED_BY_NODE=$(echo $POD_REQ | jq 'group_by(.node_name) | map({"node_name": .[0].node_name, "cpu_req": map(.cpu_req), "memory_req": map(.memory_req), "pods": map(.pods) | length})')
PODS_NULL_FILTERED=$(echo $MERGED_BY_NODE | jq 'del(.[].cpu_req[] | select(. == [null])) | del(.[].memory_req[] | select(. == [null]))')

NODES="kubectl get nodes -o json"
NODE_ALLOCATABLE=$(${NODES} | jq '[.items[] | {"node_name":.metadata.name, "allocatable":.status.allocatable}]')
COMBINED=$(echo "[" $PODS_NULL_FILTERED "," $NODE_ALLOCATABLE "]" | jq '.[0] + .[1] | group_by(.node_name) | map({"node_name":.[0].node_name, "cpu_req":.[0].cpu_req, "memory_req":.[0].memory_req, "cpu_allocatable":.[1].allocatable.cpu, "memory_allocatable": .[1].allocatable.memory})')
echo $COMBINED