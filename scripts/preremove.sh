#!/bin/sh
# Stop and disable the knocker service if systemctl is available
if command -v systemctl >/dev/null 2>&1; then
  # Check if the service exists before trying to stop it
  if systemctl list-units --full --all | grep -q "Knocker.service"; then
    systemctl stop Knocker.service 2>&1 || echo "Failed to stop Knocker service"
    systemctl disable Knocker.service 2>&1 || echo "Failed to disable Knocker service"
  fi
fi
