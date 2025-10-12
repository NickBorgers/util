FROM rclone/rclone:latest

# Copy the backup script into the container
COPY photos_backup.sh /usr/local/bin/photos_backup.sh
RUN chmod +x /usr/local/bin/photos_backup.sh

# Set default entrypoint
ENTRYPOINT ["/usr/local/bin/photos_backup.sh"]
