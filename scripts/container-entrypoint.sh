#!/bin/sh
set -eu

grocery-router bootstrap
exec grocery-router serve "$@"
