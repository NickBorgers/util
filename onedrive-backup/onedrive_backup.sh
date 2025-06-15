#!/bin/sh

set -e

# === CONFIGURATION ===
: "${ONEDRIVE_REMOTE:=onedrive}"
: "${ONEDRIVE_FOLDER:?You must set ONEDRIVE_FOLDER to the shared folder name}"
: "${BACKUP_PATH:=/mnt/backups}"
: "${BACKUP_INTERVAL_DAYS:=30}"   # Days between backups
: "${RETENTION_DAYS:=365}"        # Retention cutoff
: "${RCLONE_CONFIG:=}"
: "${CHECK_INTERVAL_SECONDS:=3600}"

export RCLONE_CONFIG="$RCLONE_CONFIG"
LAST_BACKUP_FILE="$BACKUP_PATH/.last_backup"

log() {
  echo "[`date '+%Y-%m-%d %H:%M:%S'`] $*"
}

# Find latest backup directory
find_latest_backup() {
  ls -1d "$BACKUP_PATH"/onedrive_backup_* 2>/dev/null | sort | tail -n 1
}

create_new_backup_from_latest() {
  NEW_DIR="$1"
  LATEST_DIR="$2"
  if [ -n "$LATEST_DIR" ] && [ -d "$LATEST_DIR" ]; then
    log "Copying latest backup $LATEST_DIR -> $NEW_DIR"
    cp -a "$LATEST_DIR" "$NEW_DIR"
  else
    log "No previous backup found — starting fresh at $NEW_DIR"
    mkdir -p "$NEW_DIR"
  fi
}

perform_backup() {
  TODAY=$(date +%F)
  DEST_DIR="${BACKUP_PATH}/onedrive_backup_${TODAY}"
  LATEST_DIR=$(find_latest_backup)

  create_new_backup_from_latest "$DEST_DIR" "$LATEST_DIR"

  log "Syncing OneDrive content into $DEST_DIR"
  rclone sync "${ONEDRIVE_REMOTE}:${ONEDRIVE_FOLDER}" "$DEST_DIR" \
    --copy-links --create-empty-src-dirs --log-level INFO

  echo "$TODAY" > "$LAST_BACKUP_FILE"
  log "Backup complete."
}

perform_prune() {
  log "Pruning backups older than $RETENTION_DAYS days..."
  find "$BACKUP_PATH" -maxdepth 1 -type d -name 'onedrive_backup_*' -mtime +"$RETENTION_DAYS" -exec rm -rf {} \;
  log "Prune complete."
}

# === MAIN LOOP ===
while true; do
  if [ -f "$LAST_BACKUP_FILE" ]; then
    LAST_DATE=$(cat "$LAST_BACKUP_FILE")
  else
    LAST_DATE="1970-01-01"
  fi

  NOW=$(date +%s)
  LAST=$(date -d "$LAST_DATE" +%s)
  DIFF_DAYS=$(( (NOW - LAST) / 86400 ))

  if [ "$DIFF_DAYS" -ge "$BACKUP_INTERVAL_DAYS" ]; then
    log "Backup interval met: $DIFF_DAYS days since last backup"
    perform_backup
    perform_prune
  else
    log "Not time yet: $DIFF_DAYS days since last backup"
  fi

  sleep "$CHECK_INTERVAL_SECONDS"
done

