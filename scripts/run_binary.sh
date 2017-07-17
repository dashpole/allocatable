#!/bin/sh

set +e

echo "starting shell script"

cleanup ()
{
	if ! sudo rm $BINARY; then
		echo "failed sudo rm $BINARY"
	fi
	exit 0
}

# cd into directory that lets us create and execute binaries
if ! cd /var/lib/kubelet; then
	echo "failed cd /var/lib/kubelet"
	exit 0
fi

# create the file
if ! sudo touch $BINARY; then
	echo "failed sudo touch $BINARY"
	cleanup
fi

# Allow execution and writing to the binary
if ! sudo chmod +777 $BINARY; then
	echo "failed sudo chmod +777 $BINARY"
	cleanup
fi

# download binary
if ! sudo curl https://storage.googleapis.com/allocatable/$BINARY > $BINARY; then
	echo "failed sudo curl https://storage.googleapis.com/allocatable/$BINARY > $BINARY"
	cleanup
fi

# execute binary
if ! ./$BINARY; then
	echo "failed ./$BINARY"
	cleanup
fi

cleanup