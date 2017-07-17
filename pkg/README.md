The packages are broken down as follows:  

analysis: parses and processes foreachmaster output generated by scrape_allocatable_metrics.go  

events: compiles into a binary that scrapes OOM and Eviction events for clusters, and collects cluster info  

scrape: compiles into a binary that scrapes metrics related to node allocatable, including node capacities, and aggregate pod requests

types: common structs and helper methods used to translate between kubernetes API objects, and logs.