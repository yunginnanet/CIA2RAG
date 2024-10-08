#!/bin/bash

while :; do
	_next=$(head -n 1 mullvad_relays)
	echo "switching to: $_next"
	cat mullvad_trigger >/dev/null && mullvad relay set location ${_next} >/dev/null && mullvad reconnect && sleep 0.5
	while
		! mullvad status | grep -v updated | grep 'appears'
	do sleep 1; echo -n "."; done
	echo -n "sending SIGUP..."
	if ! pkill -SIGHUP -e cia_scrape >/dev/null; then
		if ! pkill -SIGHUP -e "$(pwd | awk -F '/' '{print $NF}')" >/dev/null; then
			echo -n -e "failed :(\n"
		fi
	else echo -n -e "done\n"; fi
done
