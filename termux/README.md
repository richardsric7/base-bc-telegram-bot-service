# Running on Android (Termux)

Android 12 phones are essentially all arm64, and `android/arm64` is the one
Android target Go can cross-compile without the NDK — so that's the build
this ships. Every push to any branch builds a fresh
`base-bot-android-arm64.zip` (binary + these scripts) via GitHub Actions;
grab it from the repo's **Releases** page (tag `android-<branch>`) or from
the workflow run's artifacts.

## One-time setup

1. Install **Termux** and **Termux:Boot** from
   [F-Droid](https://f-droid.org/packages/com.termux/) — not the Play Store
   build, which is outdated and unmaintained.
2. In Termux:
   ```sh
   pkg update && pkg upgrade
   pkg install unzip
   ```
3. Download `base-bot-android-arm64.zip` on the device (via a browser, or
   `curl`/`wget` a direct release-asset URL from Termux), then:
   ```sh
   mkdir -p ~/base-bot
   cd ~/base-bot
   unzip ~/storage/downloads/base-bot-android-arm64.zip   # adjust path as needed
   chmod +x bot run.sh boot/start-bot.sh
   cp .env.example .env
   nano .env   # fill in TELEGRAM_BOT_TOKEN, ADMIN_SETUP_CODE, WALLET_ENCRYPTION_KEY, ...
   ```
4. Test it in the foreground first:
   ```sh
   ./run.sh
   ```
   Confirm in Telegram that `/start <ADMIN_SETUP_CODE>` registers you as
   admin, then `Ctrl+C` to stop.

## Auto-start on boot

```sh
mkdir -p ~/.termux/boot
cp boot/start-bot.sh ~/.termux/boot/start-base-bot.sh
chmod +x ~/.termux/boot/start-base-bot.sh
```

Then:

1. Open the **Termux:Boot** app once — it needs to run one time so Android
   registers its boot receiver. You can close it immediately after.
2. Exempt both **Termux** and **Termux:Boot** from battery optimization:
   Android Settings → Apps → (Termux / Termux:Boot) → Battery → set to
   "Unrestricted". The exact wording varies by device/OEM (some also have a
   separate "autostart" toggle to enable). Skipping this is the most common
   reason the bot silently stops after a while — Android's Doze/App Standby
   will otherwise suspend or kill it.
3. Reboot the device and confirm the bot comes back up (check
   `~/base-bot/boot.log` and `~/base-bot/bot.log`).

## Operating it

- **Logs**: `~/base-bot/bot.log` (the service's own output) and
  `~/base-bot/boot.log` (boot-script output).
- **Stop it**: `pkill -f ./bot` (the restart loop in `run.sh` will need
  killing too if you started it manually — `pkill -f run.sh`).
- **Update**: download the new release zip and `unzip -o` it over
  `~/base-bot` (this won't touch your `.env`, since that file isn't part of
  the release archive), then restart.
- **Data**: the SQLite database (`DATABASE_PATH`, default `./data/bot.db`
  relative to wherever you run it — i.e. `~/base-bot/data/bot.db`) and your
  `.env` are the only state that matters; back both up before reinstalling
  Termux or resetting the device.

This is a self-custody wallet tool running unattended on a phone — treat the
device accordingly (screen lock, encrypted storage, don't leave it somewhere
untrusted).
