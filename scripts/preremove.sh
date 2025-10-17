#!/bin/sh
if command -v knocker >/dev/null 2>&1; then
  knocker stop 2>/dev/null || true
  knocker uninstall 2>/dev/null || true
fi
