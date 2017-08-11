To build binaries: make build
To upload binaries (for use with foreachmaster): make upload
To build and upload: make

#BLAZE COMMAND for getting events:  
`blaze run cloud/kubernetes/tools:foreachmaster -- --db=prod \  
 --cmd="export BINARY=get_events; curl https://storage.googleapis.com/allocatable/run_binary.sh | sh" \  
 --shards=10 |& tee /tmp/foreachmaster.log`

To process events from the foreachmaster output, and output results into _output/eventStats.csv:
`./_output/process_events --path=/tmp/foreachmaster.log`

#BLAZE COMMAND for getting allocatable:  
`blaze run cloud/kubernetes/tools:foreachmaster -- --db=prod \  
 --cmd="export BINARY=get_allocatable_metrics; curl https://storage.googleapis.com/allocatable/run_binary.sh | sh" \  
 --shards=10 |& tee /tmp/foreachmaster.log`

To process allocatable from the foreachmaster output, and output results into _output/specificClusterStats.csv:
`./_output/allocatable_analysis --path=/tmp/foreachmaster.log`