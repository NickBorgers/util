#!/bin/sh

# === CONFIGURATION ===
: "${ONEDRIVE_REMOTE:=onedrive}"
: "${ONEDRIVE_FOLDER:?You must set ONEDRIVE_FOLDER to the shared folder name}"
: "${BACKUP_PATH:=/mnt/backups}"
: "${BACKUP_INTERVAL_DAYS:=30}"
: "${RETENTION_DAYS:=365}"
: "${RCLONE_CONFIG:=}"
: "${CHECK_INTERVAL_SECONDS:=3600}"
: "${RETRY_COOLDOWN_DAYS:=1}"

export RCLONE_CONFIG="$RCLONE_CONFIG"
LAST_BACKUP_FILE="$BACKUP_PATH/.last_backup"
LAST_ATTEMPT_FILE="$BACKUP_PATH/.last_attempt"

log() {
  echo "[`date '+%Y-%m-%d %H:%M:%S'`] $*"
}

find_latest_backup() {
  EXCLUDE="${1:-}"
  if [ -n "$EXCLUDE" ]; then
    ls -1d "$BACKUP_PATH"/onedrive_backup_* 2>/dev/null | grep -v "^${EXCLUDE}$" | sort | tail -n 1
  else
    ls -1d "$BACKUP_PATH"/onedrive_backup_* 2>/dev/null | sort | tail -n 1
  fi
}

create_new_backup_from_latest() {
  NEW_DIR="$1"
  LATEST_DIR="$2"
  if [ -n "$LATEST_DIR" ] && [ -d "$LATEST_DIR" ]; then
    log "Copying latest backup $LATEST_DIR -> $NEW_DIR (hardlinks)"
    cp -al "$LATEST_DIR" "$NEW_DIR"
  else
    log "No previous backup found — starting fresh at $NEW_DIR"
    mkdir -p "$NEW_DIR"
  fi
}

perform_backup() {
  TODAY=$(date +%F)
  DEST_DIR="${BACKUP_PATH}/onedrive_backup_${TODAY}"

  echo "$TODAY" > "$LAST_ATTEMPT_FILE"

  if [ -d "$DEST_DIR" ]; then
    log "Removing incomplete backup directory from previous attempt: $DEST_DIR"
    rm -rf "$DEST_DIR"
  fi

  LATEST_DIR=$(find_latest_backup "$DEST_DIR")

  create_new_backup_from_latest "$DEST_DIR" "$LATEST_DIR"

  log "Syncing OneDrive content into $DEST_DIR"
  rclone sync "${ONEDRIVE_REMOTE}:${ONEDRIVE_FOLDER}" "$DEST_DIR" \
    --copy-links --create-empty-src-dirs --log-level INFO
  RCLONE_EXIT=$?

  if [ "$RCLONE_EXIT" -eq 0 ]; then
    echo "$TODAY" > "$LAST_BACKUP_FILE"
    log "Backup complete."
  else
    log "ERROR: rclone sync failed with exit code $RCLONE_EXIT"
    log "Cleaning up failed backup directory: $DEST_DIR"
    rm -rf "$DEST_DIR"
  fi
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
    SHOULD_ATTEMPT=true
    if [ -f "$LAST_ATTEMPT_FILE" ]; then
      LAST_ATTEMPT_DATE=$(cat "$LAST_ATTEMPT_FILE")
      LAST_ATTEMPT=$(date -d "$LAST_ATTEMPT_DATE" +%s)
      ATTEMPT_DIFF_DAYS=$(( (NOW - LAST_ATTEMPT) / 86400 ))
      if [ "$ATTEMPT_DIFF_DAYS" -lt "$RETRY_COOLDOWN_DAYS" ]; then
        RETRY_IN=$(( RETRY_COOLDOWN_DAYS - ATTEMPT_DIFF_DAYS ))
        log "BACKUP OVERDUE: last attempt failed, next retry in $RETRY_IN day(s)"
        SHOULD_ATTEMPT=false
      fi
    fi

    if [ "$SHOULD_ATTEMPT" = true ]; then
      log "Backup interval met: $DIFF_DAYS days since last backup"
      perform_backup
      perform_prune
    fi
  else
    log "Not time yet: $DIFF_DAYS days since last backup"
  fi

  sleep "$CHECK_INTERVAL_SECONDS"
done
