#!/data/data/com.termux/files/usr/bin/sh
# Termux:Boot entrypoint. Install with:
#   mkdir -p ~/.termux/boot
#   cp termux/boot/start-bot.sh ~/.termux/boot/start-base-bot.sh
#   chmod +x ~/.termux/boot/start-base-bot.sh
# Then open the Termux:Boot app once so Android registers its boot receiver,
# and exempt both Termux and Termux:Boot from battery optimization (Android
# will otherwise kill the background process). See termux/README.md.

termux-wake-lock

BOT_DIR="$HOME/base-bot"
cd "$BOT_DIR" || exit 1

nohup ./run.sh >>boot.log 2>&1 &
