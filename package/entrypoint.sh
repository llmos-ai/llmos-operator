#!/bin/bash
set -e

exec dumb-init -- llmos-operator apiserver "${@}"
