#!/usr/bin/env sh
LLMOS_OPERATOR_VERSION=${CHART_VERSION}
DEFAULT_STATIC_CHARTS_PATH=/var/lib/rancher/k3s/server/static/charts
LLMOS_CHART_PATH=/var/lib/llmos/charts
LLMOS_SYSTEM_NS=llmos-system

if [ -z "$LLMOS_OPERATOR_VERSION" ]; then
  echo "LLMOS_OPERATOR_VERSION is not set. Exiting."
  exit 1
else
  echo "LLMOS_OPERATOR_VERSION is set to '$LLMOS_OPERATOR_VERSION'"
fi


if [ ! -d "$DEFAULT_STATIC_CHARTS_PATH" ]; then
  mkdir -p "$DEFAULT_STATIC_CHARTS_PATH"
fi

if [ ! -d "$LLMOS_CHART_PATH" ]; then
  mkdir -p "$LLMOS_CHART_PATH"
fi

cp -rf ./llmos-crd*.tgz ./llmos-operator*.tgz "$DEFAULT_STATIC_CHARTS_PATH"

# Check if the namespace exists
if kubectl get ns "$LLMOS_SYSTEM_NS" >/dev/null 2>&1; then
  echo "Namespace '$LLMOS_SYSTEM_NS' already exists."
else
  echo "Namespace '$LLMOS_SYSTEM_NS' does not exist. Creating it now..."
  cat <<EOF | kubectl create -f -
apiVersion: v1
kind: Namespace
metadata:
  name: llmos-system
EOF
  echo "Namespace '$LLMOS_SYSTEM_NS' created."
fi

# Add llmos-crd HelmChart
cat <<EOF > "$LLMOS_CHART_PATH"/llmos-crd-chart.yaml
---
apiVersion: helm.cattle.io/v1
kind: HelmChart
metadata:
  name: llmos-crd
  namespace: llmos-system
spec:
  chart: https://%{KUBERNETES_API}%/static/charts/llmos-crd-$LLMOS_OPERATOR_VERSION.tgz
  failurePolicy: manual
  valuesContent: |-
    $LLMOS_CRD_VALUES
EOF

# Add llmos-operator HelmChart
cat <<EOF > "$LLMOS_CHART_PATH"/llmos-operator-chart.yaml
---
apiVersion: helm.cattle.io/v1
kind: HelmChart
metadata:
  name: llmos-operator
  namespace: llmos-system
spec:
  chart: https://%{KUBERNETES_API}%/static/charts/llmos-operator-$LLMOS_OPERATOR_VERSION.tgz
  valuesContent: |-
    $LLMOS_VALUES
EOF

kubectl apply -f "${LLMOS_CHART_PATH}"/llmos-crd-chart.yaml
kubectl apply -f "${LLMOS_CHART_PATH}"/llmos-operator-chart.yaml

# Wait for the helm chart install to be ready first
echo "Waiting for the llmos-crd and llmos-operator helm charts to be ready..."
kubectl wait --for=condition=complete --timeout=30s job/helm-install-llmos-crd -n llmos-system
kubectl wait --for=condition=complete --timeout=30s job/helm-install-llmos-operator -n llmos-system

# Wait for the llmos-operator pods to be ready
echo "Waiting for llmos-operator pods to be ready..."
sleep 3 # wait for a few seconds for the helm chart's deployment to show up
kubectl wait --for=condition=ready --timeout=600s pod -l app.kubernetes.io/name=llmos-operator -n llmos-system
