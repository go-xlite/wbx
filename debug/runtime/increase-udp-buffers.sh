#!/bin/bash

# Script to increase UDP buffer sizes for better HTTP/3 performance
# Must be run as root or with sudo

# Check if running as root
if [ "$EUID" -ne 0 ]; then
  echo "Please run as root or with sudo"
  exit 1
fi

echo "Setting UDP buffer sizes for improved HTTP/3 performance..."

# Temporarily set values for the current session
echo "Setting temporary values for current session..."
sysctl -w net.core.rmem_max=7500000
sysctl -w net.core.wmem_max=7500000

# Make changes permanent
echo "Making changes permanent in /etc/sysctl.conf..."

# Check if entries already exist
if grep -q "net.core.rmem_max" /etc/sysctl.conf; then
  # Update existing entries
  sed -i 's/net.core.rmem_max=.*/net.core.rmem_max=7500000/g' /etc/sysctl.conf
else
  # Add new entries
  echo "net.core.rmem_max=7500000" >> /etc/sysctl.conf
fi

if grep -q "net.core.wmem_max" /etc/sysctl.conf; then
  sed -i 's/net.core.wmem_max=.*/net.core.wmem_max=7500000/g' /etc/sysctl.conf
else
  echo "net.core.wmem_max=7500000" >> /etc/sysctl.conf
fi

echo "UDP buffer sizes have been increased."
echo "You can verify with: sysctl net.core.rmem_max net.core.wmem_max"
echo "Changes will persist after reboot."

# Apply changes without requiring reboot
sysctl -p

echo "Done!"
