The packages are broken down as follows:  

allocatable: scrape allocatable metrics using foreachmaster, and process output to produce allocatable stats  

events: scrape oom and eviction events using foreachmaster, and process output
to produce stats on disruptive events 

types: common structs and helper methods used to translate between kubernetes API objects, and logs.
