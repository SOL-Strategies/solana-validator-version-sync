#!/bin/bash
set -euo pipefail

case $1 in
  build)
    shift 1
    echo "building $*"
    sleep 1
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
    echo done
    ;;
  monitor)
    shift 1
    echo "wating for healthy validator at $*/health..."
    echo done
    ;;
  notify)
    shift 1
    echo "sending slack notification version change $*..."
    echo done
    ;;
esac