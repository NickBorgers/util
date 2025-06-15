FROM rclone/rclone:latest

# Copy the backup script into the container
COPY onedrive_backup.sh /usr/local/bin/onedrive_backup.sh
RUN chmod +x /usr/local/bin/onedrive_backup.sh

# Set default entrypoint
ENTRYPOINT ["/usr/local/bin/onedrive_backup.sh"]
