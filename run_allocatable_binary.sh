#!/bin/sh

#BLAZE COMMAND:
# blaze run cloud/kubernetes/tools:foreachmaster -- --db=prod \
# --cmd="curl https://storage.googleapis.com/allocatable/run_allocatable_binary.sh | sh" \
# --shards=50 |& tee /tmp/foreachmaster.log
set +e

echo "starting shell script"

cleanup ()
{
	if ! sudo rm get_allocatable_metrics; then
		echo "failed sudo rm get_allocatable_metrics"
	fi
	exit 0
}

# cd into directory that lets us create and execute binaries
if ! cd /var/lib/kubelet; then
	echo "failed cd /var/lib/kubelet"
	exit 0
fi

# create the file
if ! sudo touch get_allocatable_metrics; then
	echo "failed sudo touch get_allocatable_metrics"
	cleanup
fi

# Allow execution and writing to the binary
if ! sudo chmod +777 get_allocatable_metrics; then
	echo "failed sudo chmod +777 get_allocatable_metrics"
	cleanup
fi

# download binary
if ! sudo curl https://storage.googleapis.com/allocatable/get_allocatable_metrics > get_allocatable_metrics; then
	echo "failed sudo curl https://storage.googleapis.com/allocatable/get_allocatable_metrics > get_allocatable_metrics"
	cleanup
fi

# execute binary
if ! ./get_allocatable_metrics; then
	echo "failed ./get_allocatable_metrics"
	cleanup
fi

cleanup