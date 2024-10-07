#!/bin/bash

if ! mullvad status; then
	echo "mullvad not connected or installed"
	exit 1
fi

if ! ls mullvad_relays; then
	echo "creating mullvad_relays fifo"
	mkfifo mullvad_relays || exit 1
fi

if ! ls mullvad_trigger; then
	echo "creating mullvad_trigger fifo"
	mkfifo mullvad_trigger || exit 1
fi

./mullvad_relays.sh &

./mullvad_switch.sh &
