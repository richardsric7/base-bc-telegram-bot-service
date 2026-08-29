#!/data/data/com.termux/files/usr/bin/bash
# Restart-loop wrapper for running the bot binary under Termux. Keeps it
# running across crashes/RPC hiccups. Pair with boot/start-bot.sh (installed
# to ~/.termux/boot/) so it also survives device reboots.
set -u
cd "$(dirname "$0")"

if [ ! -f .env ]; then
	echo "Missing .env — copy .env.example to .env and fill it in first." >&2
	exit 1
fi
if [ ! -x ./bot ]; then
	echo "Missing or non-executable ./bot binary." >&2
	exit 1
fi

while true; do
	./bot 2>&1 | tee -a bot.log
	echo "$(date '+%Y-%m-%d %H:%M:%S'): bot exited, restarting in 5s..." >>bot.log
	sleep 5
done
