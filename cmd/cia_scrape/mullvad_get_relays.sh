#!/bin/bash

while :; do mullvad relay list | grep -A 500 USA | grep WireGuard | shuf | awk '{print $1}' | while read -r server; do
	_loc="$(echo "${server}" | awk -F '-' '{print $2}')" && _id="$(
		echo "${server}" | awk '{print $1}'
	)"
	echo "us $_loc $_id"
done >>mullvad_relays; done
