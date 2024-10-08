#!/bin/bash

function _kill() {
	pkill -i -e mullvad_ || return 1
}

_hr="-----------------------------------\n"

echo -n -e "${_hr}\n"

_kill 2>/dev/null

echo -e "\n"

if ! _status="$(mullvad status)"; then
	echo "mullvad not connected or installed"
	exit 1
else
	echo -e "$_status" | sed 's|IPv4|\nIPv4|g'
fi

echo -e "\n${_hr}"

if ! ls mullvad_relays >/dev/null; then
	echo "creating mullvad_relays fifo"
	/ mkfifo mullvad_relays || exit 1
fi

if ! ls mullvad_trigger >/dev/null; then
	echo "creating mullvad_trigger fifo"
	mkfifo mullvad_trigger || exit 1
fi

./mullvad_get_relays.sh &
./mullvad_switch.sh &
if ! pgrep -f mullvad_get_relays.sh && pgrep -f mullvad_switch.sh; then
	_kill && echo "failed" && exit 1
else
	echo -e "\nscripts started."
fi
