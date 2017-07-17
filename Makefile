all: 
	make build
	make upload

build:
	go build --ldflags '-linkmode external -extldflags "-static"' -o _output/get_allocatable_metrics pkg/scrape/scrape_allocatable_metrics.go

	go build --ldflags '-linkmode external -extldflags "-static"' -o _output/get_events pkg/events/scrape_events.go

	go build --ldflags '-linkmode external -extldflags "-static"' -o _output/allocatable_analysis pkg/analysis/allocatable_analysis.go

upload: 
	gsutil cp ./scripts/run_binary.sh gs://allocatable
	gsutil acl ch -u AllUsers:R gs://allocatable/run_binary.sh
	gsutil setmeta -h "Cache-Control: private, max-age=0" gs://allocatable/run_binary.sh

	gsutil cp _output/get_events gs://allocatable
	gsutil acl ch -u AllUsers:R gs://allocatable/get_events
	gsutil setmeta -h "Cache-Control: private, max-age=0" gs://allocatable/get_events

	gsutil cp _output/get_allocatable_metrics gs://allocatable
	gsutil acl ch -u AllUsers:R gs://allocatable/get_allocatable_metrics
	gsutil setmeta -h "Cache-Control: private, max-age=0" gs://allocatable/get_allocatable_metrics