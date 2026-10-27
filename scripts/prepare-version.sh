#!/bin/sh
set -eu

APP_VERSION=${APP_VERSION:-dev}
APP_VERSION=${APP_VERSION#v}
APP_VERSION=$(printf '%s' "$APP_VERSION" | tr '/' '-')
printf '%s' "$APP_VERSION" > cmd/version.txt
