#!/bin/sh

cleanup() {
  echo "Caught SIGTERM, exiting cleanly..."
  exit 0
}

trap cleanup TERM

# Main logic
echo "Updating packages in case you want to install something else"
apt update
echo "Sleeping forever so this container keeps running for you to exec into it"
sleep infinity &
wait $!

