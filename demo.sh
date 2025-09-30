#!/bin/bash
set -euo pipefail

case $1 in
  build)
    shift 1
    echo "building $*"
    sleep 2
    echo "oops! stderr msg, but still building..." >&2
    sleep 1
    echo "done"
    ;;
  install)
    shift 1
    echo "installing $*"
    sleep 1
    echo "done"
    ;;
  restart)
    shift 1
    echo "restarting $*..."
    sleep 1
    echo done
    ;;
  monitor)
    shift 1
    echo "waiting for healthy validator at $*/health..."
    sleep 5
    echo done
    ;;
  notify)
    shift 1
    echo "sending slack notification for version change $*..."
    sleep 1
    echo done
    ;;
esac