# OneDrive Backup Container

This is really just the rclone base image + a script ChatGPT wrote for me.

## How to run it
You invoke it like this:
```
docker run \
	--rm \
	-it \
	--mount type=bind,src=./rclone.conf,dst=/etc/rclone/ \
	--mount type=bind,src=./temp/,dst=/mnt/backups/ \
	-e RCLONE_CONFIG="/etc/rclone/rclone.conf" \
	-e ONEDRIVE_FOLDER=/ \
	nickborgers/onedrive-backup
```

You'll need to modify all of those parameters for you situation:
  * `rclone.conf` must be generated the usual way (with rclone) and include a onedrive configuration called `onedrive` (or you can pass another ENV_VAR to script)
  * Specify whatever destination you want, the script will write to `/mnt/backups`
  * `ONEDRIVE_FOLDER` depends on what you want to backup; this example is `/` so everything in your OneDrive
    ** Shared folders are tricky IMO, you can create a shortcut of them into "your OneDrive" which will cause them to be recursively backed-up

## Behaviors
This container is meant to be run forever. I'm using a hosting solution which probably would allow me to have a cron job but I'm not going that route. A `bash` script sleeping doesn't consume a significant amount of resources, so is good enough.

Every hour it will check if it should make a new backup, and if `BACKUP_INTERVAL` has elpased it will copy the last backup to become a new backup. Then it runs `rclone` against the newly created directory which was based on the last backup.

Why do the copy? Just trying to reduce how much downloading actually occurs; rclone seems to sync it pretty well. I do want a complete separate backup because of the why for this.

## Why do this?
Specifically, why backup a cloud storage solution a vendor is promising you has HA and its own backups? It even has ransomware protection so if your data gets hosed that way they can help you recover it.

Honestly this is more about protecting from human error, but also it is *possible* MSFT could screw up bad enough to lose everything. In a way it's still a single copy, and OneDrive clients might go delete the local copies of the server told them to.

Storage is cheap, family memories are not.

Note: the locally saved copy is the backup, and is excluded from my solutions for backing up local copies elsewhere. This is the end of the backup chain.

