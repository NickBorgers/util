#!/bin/sh

set -e

# === CONFIGURATION ===
: "${GDRIVE_REMOTE:=gdrive}"
: "${LOCAL_PHOTOS_PATH:?You must set LOCAL_PHOTOS_PATH to the local photos folder path}"
: "${GDRIVE_FOLDER:=Photos}"        # Google Drive destination folder
: "${BACKUP_INTERVAL_HOURS:=6}"     # Hours between backups (default 6 hours)
: "${RCLONE_CONFIG:=}"
: "${CHECK_INTERVAL_SECONDS:=1800}" # Check every 30 minutes (1800 seconds)
: "${USE_SYNC_MODE:=false}"         # Set to "true" to enable sync mode (deletes remote files)

export RCLONE_CONFIG="$RCLONE_CONFIG"
LAST_BACKUP_FILE="/tmp/.last_photos_backup"

log() {
  echo "[`date '+%Y-%m-%d %H:%M:%S'`] $*"
}

# Check if it's time for backup
should_backup() {
  if [ ! -f "$LAST_BACKUP_FILE" ]; then
    return 0  # No previous backup, should backup
  fi
  
  LAST_BACKUP=$(cat "$LAST_BACKUP_FILE")
  NOW=$(date +%s)
  DIFF_HOURS=$(( (NOW - LAST_BACKUP) / 3600 ))
  
  if [ "$DIFF_HOURS" -ge "$BACKUP_INTERVAL_HOURS" ]; then
    return 0  # Should backup
  else
    return 1  # Should not backup
  fi
}

perform_backup() {
  # Determine sync vs copy mode
  if [ "$USE_SYNC_MODE" = "true" ]; then
    SYNC_MODE="sync"
    log "Starting photos SYNC from $LOCAL_PHOTOS_PATH to ${GDRIVE_REMOTE}:${GDRIVE_FOLDER} (WARNING: This will DELETE remote files not present locally)"
  else
    SYNC_MODE="copy"
    log "Starting photos COPY from $LOCAL_PHOTOS_PATH to ${GDRIVE_REMOTE}:${GDRIVE_FOLDER} (safe mode: will not delete remote files)"
  fi
  
  # Copy/sync local photos to Google Drive
  rclone "$SYNC_MODE" "$LOCAL_PHOTOS_PATH" "${GDRIVE_REMOTE}:${GDRIVE_FOLDER}" \
    --create-empty-src-dirs \
    --log-level INFO \
    --progress \
    --stats=30s \
    --retries=3 \
    --low-level-retries=10
  
  # Update last backup timestamp
  date +%s > "$LAST_BACKUP_FILE"
  log "Photos backup complete."
}

# Validate configuration
validate_config() {
  if [ ! -d "$LOCAL_PHOTOS_PATH" ]; then
    log "ERROR: Local photos path does not exist: $LOCAL_PHOTOS_PATH"
    exit 1
  fi
  
  if [ -z "$RCLONE_CONFIG" ]; then
    log "ERROR: RCLONE_CONFIG environment variable must be set"
    exit 1
  fi
  
  if [ ! -f "$RCLONE_CONFIG" ]; then
    log "ERROR: Rclone config file does not exist: $RCLONE_CONFIG"
    exit 1
  fi
  
  # Test rclone connection
  log "Testing rclone connection to ${GDRIVE_REMOTE}..."
  if ! rclone lsd "${GDRIVE_REMOTE}:" >/dev/null 2>&1; then
    log "ERROR: Cannot connect to ${GDRIVE_REMOTE}. Please check your rclone configuration."
    exit 1
  fi
  
  log "Configuration validation successful."
}

# === MAIN LOOP ===
log "Starting photos backup service..."
log "Local photos path: $LOCAL_PHOTOS_PATH"
log "Google Drive remote: ${GDRIVE_REMOTE}:${GDRIVE_FOLDER}"
log "Backup interval: $BACKUP_INTERVAL_HOURS hours"
log "Check interval: $((CHECK_INTERVAL_SECONDS / 60)) minutes"
log "Sync mode: $([ "$USE_SYNC_MODE" = "true" ] && echo "ENABLED (will delete remote files)" || echo "DISABLED (safe copy mode)")"

validate_config

# Perform initial backup if no previous backup exists
if [ ! -f "$LAST_BACKUP_FILE" ]; then
  log "No previous backup found. Performing initial backup..."
  perform_backup
fi

while true; do
  if should_backup; then
    LAST_BACKUP=$(cat "$LAST_BACKUP_FILE" 2>/dev/null || echo "0")
    HOURS_SINCE=$(( ( $(date +%s) - LAST_BACKUP ) / 3600 ))
    log "Backup interval met: $HOURS_SINCE hours since last backup"
    perform_backup
  else
    LAST_BACKUP=$(cat "$LAST_BACKUP_FILE" 2>/dev/null || echo "0")
    HOURS_SINCE=$(( ( $(date +%s) - LAST_BACKUP ) / 3600 ))
    log "Not time yet: $HOURS_SINCE hours since last backup (interval: $BACKUP_INTERVAL_HOURS hours)"
  fi
  
  log "Sleeping for $((CHECK_INTERVAL_SECONDS / 60)) minutes..."
  sleep "$CHECK_INTERVAL_SECONDS"
done
