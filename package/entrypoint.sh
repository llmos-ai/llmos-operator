#!/bin/bash
set -e

# Unified entrypoint script for llmos-operator
# Can be used for apiserver, webhook, or downloader modes
# Usage:
#   - Set LLMOS_MODE environment variable (apiserver, webhook, download)
#   - Or pass mode as first argument
#   - Defaults to apiserver if no mode specified

# Determine the mode
MODE="${LLMOS_MODE:-${1:-apiserver}}"

# Shift arguments if mode was passed as first argument
if [ "$1" = "apiserver" ] || [ "$1" = "webhook" ] || [ "$1" = "download" ]; then
    shift
fi

case "$MODE" in
    "apiserver")
        exec tini -- llmos-operator apiserver "${@}"
        ;;
    "webhook")
        exec tini -- llmos-operator webhook "${@}"
        ;;
    "download")
        exec tini -- llmos-operator download "${@}"
        ;;
    *)
        echo "Error: Invalid mode '$MODE'. Valid modes are: apiserver, webhook, download"
        exit 1
        ;;
esac