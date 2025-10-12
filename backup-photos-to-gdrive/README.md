# Photos Backup to Google Drive Container

This container continuously backs up a local photos folder to Google Drive using rclone. It's designed to run forever with minimal resource usage, performing incremental backups at configurable intervals.

## How to run it

You invoke it like this:
```bash
docker run \
  --rm \
  --mount type=bind,src=./rclone.conf,dst=/etc/rclone/rclone.conf \
  --mount type=bind,src=/path/to/your/photos,dst=/mnt/photos \
  -e RCLONE_CONFIG="/etc/rclone/rclone.conf" \
  -e LOCAL_PHOTOS_PATH="/mnt/photos" \
  -e GDRIVE_REMOTE="gdrive" \
  -e GDRIVE_FOLDER="Photos" \
  -e BACKUP_INTERVAL_HOURS="6" \
  backup-photos-to-gdrive
```

You'll need to modify these parameters for your situation:
  * `rclone.conf` must be generated using rclone and include a Google Drive configuration (default name: `gdrive`)
  * Mount your local photos folder to `/mnt/photos` (or another path and update `LOCAL_PHOTOS_PATH`)
  * `GDRIVE_FOLDER` specifies the destination folder in Google Drive (default: "Photos")
  * `BACKUP_INTERVAL_HOURS` sets how often to backup (default: 6 hours)

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `LOCAL_PHOTOS_PATH` | *required* | Path to the local photos folder inside the container |
| `GDRIVE_REMOTE` | `gdrive` | Name of the rclone Google Drive remote |
| `GDRIVE_FOLDER` | `Photos` | Destination folder in Google Drive |
| `BACKUP_INTERVAL_HOURS` | `6` | Hours between backups |
| `CHECK_INTERVAL_SECONDS` | `1800` | How often to check if backup is needed (30 minutes) |
| `USE_SYNC_MODE` | `false` | Set to `true` to enable sync mode (deletes remote files not present locally) |
| `RCLONE_CONFIG` | *required* | Path to rclone configuration file |

## Behaviors

This container is designed to run continuously with minimal resource usage. The bash script sleeps between checks, consuming virtually no system resources while waiting.

**Key Features:**
- **Safe Copy Mode**: By default, only adds/updates files without deleting remote files
- **Optional Sync Mode**: Can enable sync mode to maintain exact duplicate (deletes remote files)
- **Incremental Transfer**: Only transfers new or modified files
- **Automatic Retries**: Built-in retry logic for failed transfers
- **Progress Monitoring**: Shows transfer progress and statistics
- **Configuration Validation**: Validates setup before starting backups

**Backup Process:**
1. Every 30 minutes, checks if backup interval has elapsed
2. If time for backup, copies/syncs local photos to Google Drive
3. Uses rclone's copy command by default (safe mode)
4. Can optionally use rclone's sync command (deletes remote files)
5. Logs all activities with timestamps

## Setup Instructions

### 1. Configure rclone for Google Drive

First, set up rclone with Google Drive access:

```bash
rclone config
```

Create a new remote called `gdrive` (or use a different name and update `GDRIVE_REMOTE`). Follow the authentication flow to grant access to your Google Drive.

### 2. Test the Configuration

Verify your rclone setup works:

```bash
rclone lsd gdrive:
```

### 3. Build the Container

```bash
# Build locally
docker build -t backup-photos-to-gdrive ./backup-photos-to-gdrive

# Or use the pre-built image from GitHub Container Registry
docker pull ghcr.io/your-username/util/backup-photos-to-gdrive:latest
```

### 4. Run the Container

```bash
docker run \
  -d \
  --name photos-backup \
  --restart unless-stopped \
  --mount type=bind,src=./rclone.conf,dst=/etc/rclone/rclone.conf \
  --mount type=bind,src=/path/to/your/photos,dst=/mnt/photos \
  -e RCLONE_CONFIG="/etc/rclone/rclone.conf" \
  -e LOCAL_PHOTOS_PATH="/mnt/photos" \
  -e BACKUP_INTERVAL_HOURS="6" \
  backup-photos-to-gdrive
```

## Sync vs Copy Mode

**Copy Mode (Default - Safe)**
- Uses `rclone copy` - only adds/updates files
- Never deletes files from Google Drive
- Safe for continuous operation
- Remote files persist even if deleted locally

**Sync Mode (Optional - Destructive)**
- Uses `rclone sync` - maintains exact duplicate
- Deletes remote files not present locally
- Enable with `USE_SYNC_MODE=true`
- Use carefully - can permanently delete remote files

To enable sync mode:
```bash
docker run \
  -d \
  --name photos-backup \
  --restart unless-stopped \
  --mount type=bind,src=./rclone.conf,dst=/etc/rclone/rclone.conf \
  --mount type=bind,src=/path/to/your/photos,dst=/mnt/photos \
  -e LOCAL_PHOTOS_PATH="/mnt/photos" \
  -e USE_SYNC_MODE="true" \
  backup-photos-to-gdrive
```

## Monitoring

Check container logs to monitor backup activity:

```bash
docker logs -f photos-backup
```

Example log output:
```
[2024-01-15 10:30:00] Starting photos backup service...
[2024-01-15 10:30:00] Local photos path: /mnt/photos
[2024-01-15 10:30:00] Google Drive remote: gdrive:Photos
[2024-01-15 10:30:00] Backup interval: 6 hours
[2024-01-15 10:30:00] Check interval: 30 minutes
[2024-01-15 10:30:00] Sync mode: DISABLED (safe copy mode)
[2024-01-15 10:30:00] Configuration validation successful.
[2024-01-15 10:30:00] Starting photos COPY from /mnt/photos to gdrive:Photos (safe mode: will not delete remote files)
[2024-01-15 10:32:15] Photos backup complete.
[2024-01-15 10:32:15] Sleeping for 30 minutes...
```

## Resource Usage

- **CPU**: Near zero when idle, brief spikes during sync operations
- **Memory**: Minimal (~10-50MB depending on rclone version)
- **Network**: Only during active sync operations
- **Storage**: No additional storage used (direct sync to Google Drive)

## Troubleshooting

### Common Issues

1. **Authentication Errors**: Ensure rclone config is properly mounted and Google Drive access is configured
2. **Permission Denied**: Check that the mounted photos folder has proper read permissions
3. **Network Issues**: The script includes retry logic, but persistent network issues may require manual intervention

### Debug Mode

For more verbose logging, you can modify the script to use `--log-level DEBUG` instead of `--log-level INFO`.

## Why This Approach?

- **Simple**: No complex backup strategies or versioning - just a straightforward sync
- **Efficient**: Only transfers what's changed, reducing bandwidth and time
- **Reliable**: Built on rclone's proven sync capabilities with retry logic
- **Low Resource**: Minimal system impact while running continuously
- **Flexible**: Easy to adjust backup frequency and destination settings

Perfect for scenarios where you want a simple, reliable way to keep Google Drive in sync with a local photos collection without the complexity of traditional backup solutions.
