#!/bin/bash

gsutil cp ./run_allocatable_binary.sh gs://allocatable
gsutil acl ch -u AllUsers:R gs://allocatable/run_allocatable_binary.sh
gsutil setmeta -h "Cache-Control: private, max-age=0" gs://allocatable/run_allocatable_binary.sh

go build --ldflags '-linkmode external -extldflags "-static"' -o get_allocatable_metrics allocatable_metrics.go

gsutil cp ./get_allocatable_metrics gs://allocatable
gsutil acl ch -u AllUsers:R gs://allocatable/get_allocatable_metrics
gsutil setmeta -h "Cache-Control: private, max-age=0" gs://allocatable/get_allocatable_metrics