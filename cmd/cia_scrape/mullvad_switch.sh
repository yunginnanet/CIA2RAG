#!/bin/bash

while :; do
	cat mullvad_trigger >/dev/null && mullvad relay set location "$(head -n 1 mullvad_relays >/dev/null)" && mullvad reconnect && sleep 0.5
	while
		! mullvad status | grep -v
		updated | grep 'appears'
	do sleep 1; done
done
