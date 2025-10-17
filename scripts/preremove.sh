#!/bin/sh
# Stop and uninstall the knocker service
# This script works on Linux (systemd), macOS (launchd), and other systems

# First try to use the knocker CLI commands if available
if command -v knocker >/dev/null 2>&1; then
  echo "Stopping Knocker service..."
  knocker stop 2>&1 || echo "Warning: Failed to stop Knocker service (may not be running)"
  
  echo "Uninstalling Knocker service..."
  knocker uninstall 2>&1 || echo "Warning: Failed to uninstall Knocker service (may not be installed)"
else
  # Fallback to platform-specific service management
  echo "knocker command not found, trying platform-specific service management..."
  
  # Try systemd (Linux)
  if command -v systemctl >/dev/null 2>&1; then
    if systemctl list-units --full --all | grep -q "Knocker.service"; then
      echo "Stopping Knocker service via systemctl..."
      systemctl stop Knocker.service 2>&1 || echo "Warning: Failed to stop Knocker service"
      systemctl disable Knocker.service 2>&1 || echo "Warning: Failed to disable Knocker service"
    fi
  # Try launchd (macOS)
  elif command -v launchctl >/dev/null 2>&1; then
    # macOS service management
    SERVICE_PLIST="/Library/LaunchDaemons/Knocker.plist"
    if [ -f "$SERVICE_PLIST" ]; then
      echo "Stopping Knocker service via launchctl..."
      launchctl unload "$SERVICE_PLIST" 2>&1 || echo "Warning: Failed to unload Knocker service"
    fi
  fi
fi
