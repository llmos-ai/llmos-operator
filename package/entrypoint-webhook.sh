#!/bin/bash
set -e

exec dumb-init -- llmos-controller webhook "${@}"
