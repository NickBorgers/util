#!/bin/sh

# Requires GNU/BusyBox coreutils (date -d, cp -al) — runs in Alpine-based rclone container

# === CONFIGURATION ===
: "${ONEDRIVE_REMOTE:=onedrive}"
: "${ONEDRIVE_FOLDER:?You must set ONEDRIVE_FOLDER to the shared folder name}"
: "${BACKUP_PATH:=/mnt/backups}"
: "${BACKUP_INTERVAL_DAYS:=30}"
: "${RETENTION_DAYS:=1825}"
: "${RCLONE_CONFIG:=}"
: "${CHECK_INTERVAL_SECONDS:=3600}"
: "${RETRY_COOLDOWN_DAYS:=1}"

export RCLONE_CONFIG="$RCLONE_CONFIG"
mkdir -p "$BACKUP_PATH" || { echo "FATAL: cannot create $BACKUP_PATH" >&2; exit 1; }
LAST_BACKUP_FILE="$BACKUP_PATH/.last_backup"
LAST_ATTEMPT_FILE="$BACKUP_PATH/.last_attempt"

log() {
  echo "[`date '+%Y-%m-%d %H:%M:%S'`] $*"
}

write_state_file() {
  TARGET="$1"
  CONTENT="$2"
  TMPFILE="${TARGET}.tmp"
  if echo "$CONTENT" > "$TMPFILE" && mv "$TMPFILE" "$TARGET"; then
    return 0
  fi
  rm -f "$TMPFILE"
  return 1
}

read_epoch_file() {
  FILE="$1"
  if [ -f "$FILE" ]; then
    VALUE=$(cat "$FILE")
    if echo "$VALUE" | grep -qE '^[0-9]+$'; then
      echo "$VALUE"
      return 0
    fi
    log "WARNING: corrupt state file $FILE (contents: '$VALUE'), treating as absent" >&2
  fi
  echo "0"
}

read_date_file() {
  FILE="$1"
  if [ -f "$FILE" ]; then
    VALUE=$(cat "$FILE")
    if echo "$VALUE" | grep -qE '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' && date -d "$VALUE" +%s >/dev/null 2>&1; then
      echo "$VALUE"
      return 0
    fi
    log "WARNING: corrupt state file $FILE (contents: '$VALUE'), treating as absent" >&2
  fi
  echo "1970-01-01"
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
    if ! cp -al "$LATEST_DIR" "$NEW_DIR"; then
      log "ERROR: failed to create hardlink snapshot from $LATEST_DIR"
      rm -rf "$NEW_DIR"
      return 1
    fi
  else
    log "No previous backup found — starting fresh at $NEW_DIR"
    mkdir -p "$NEW_DIR"
  fi
}

perform_backup() {
  TODAY=$(date +%F)
  NOW_EPOCH=$(date +%s)
  DEST_DIR="${BACKUP_PATH}/onedrive_backup_${TODAY}"

  if ! write_state_file "$LAST_ATTEMPT_FILE" "$NOW_EPOCH"; then
    log "ERROR: cannot write $LAST_ATTEMPT_FILE — aborting backup to prevent retry loop"
    return 1
  fi

  if [ -d "$DEST_DIR" ]; then
    log "Removing incomplete backup directory from previous attempt: $DEST_DIR"
    rm -rf "$DEST_DIR"
  fi

  LATEST_DIR=$(find_latest_backup "$DEST_DIR")

  if ! create_new_backup_from_latest "$DEST_DIR" "$LATEST_DIR"; then
    return 1
  fi

  log "Syncing OneDrive content into $DEST_DIR"
  rclone sync "${ONEDRIVE_REMOTE}:${ONEDRIVE_FOLDER}" "$DEST_DIR" \
    --copy-links --create-empty-src-dirs --log-level INFO
  RCLONE_EXIT=$?

  if [ "$RCLONE_EXIT" -eq 0 ]; then
    if ! write_state_file "$LAST_BACKUP_FILE" "$TODAY"; then
      log "ERROR: backup succeeded but cannot write $LAST_BACKUP_FILE — keeping backup directory"
      return 1
    fi
    log "Backup complete."
    return 0
  else
    log "ERROR: rclone sync failed with exit code $RCLONE_EXIT"
    log "Cleaning up failed backup directory: $DEST_DIR"
    rm -rf "$DEST_DIR"
    return 1
  fi
}

perform_prune() {
  log "Pruning backups older than $RETENTION_DAYS days..."
  find "$BACKUP_PATH" -maxdepth 1 -type d -name 'onedrive_backup_*' -mtime +"$RETENTION_DAYS" -exec rm -rf {} \;
  log "Prune complete."
}

# === MAIN LOOP ===
while true; do
  LAST_DATE=$(read_date_file "$LAST_BACKUP_FILE")

  NOW=$(date +%s)
  LAST=$(date -d "$LAST_DATE" +%s)
  DIFF_DAYS=$(( (NOW - LAST) / 86400 ))

  if [ "$DIFF_DAYS" -ge "$BACKUP_INTERVAL_DAYS" ]; then
    SHOULD_ATTEMPT=true
    LAST_ATTEMPT_EPOCH=$(read_epoch_file "$LAST_ATTEMPT_FILE")
    if [ "$LAST_ATTEMPT_EPOCH" -gt "$NOW" ]; then
      log "WARNING: .last_attempt is in the future, ignoring"
      LAST_ATTEMPT_EPOCH=0
    fi
    if [ "$LAST_ATTEMPT_EPOCH" -gt 0 ]; then
      COOLDOWN_SECONDS=$(( RETRY_COOLDOWN_DAYS * 86400 ))
      ELAPSED=$(( NOW - LAST_ATTEMPT_EPOCH ))
      if [ "$ELAPSED" -lt "$COOLDOWN_SECONDS" ]; then
        REMAINING_HOURS=$(( (COOLDOWN_SECONDS - ELAPSED) / 3600 ))
        log "BACKUP OVERDUE: last attempt failed, next retry in ~${REMAINING_HOURS}h"
        SHOULD_ATTEMPT=false
      fi
    fi

    if [ "$SHOULD_ATTEMPT" = true ]; then
      log "Backup interval met: $DIFF_DAYS days since last backup"
      if perform_backup; then
        perform_prune
      fi
    fi
  else
    log "Not time yet: $DIFF_DAYS days since last backup"
  fi

  sleep "$CHECK_INTERVAL_SECONDS"
done
