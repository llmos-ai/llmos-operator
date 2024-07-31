#!/usr/bin/env sh
./helm upgrade --create-namespace --install --values "$LLMOS_CRD_VALUES" -n llmos-system --wait llmos-crd ./llmos-crd*.tgz
./helm upgrade --create-namespace --install --values "$LLMOS_VALUES" -n llmos-system --wait llmos-operator ./llmos-operator*.tgz
