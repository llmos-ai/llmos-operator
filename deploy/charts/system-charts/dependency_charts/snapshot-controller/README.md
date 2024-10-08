# Snapshotter Controller

The [CSI Snapshotter](https://github.com/kubernetes-csi/external-snapshotter) is part of the Kubernetes implementation of the Container Storage Interface (CSI).

The volume snapshot feature supports CSI v1.0 and higher. It was introduced as an alpha feature in Kubernetes v1.12, promoted to beta in v1.17, and reached general availability (GA) in v1.20.

## Introduction

This chart installs the CSI Snapshotter Controller & Webhook on the [LLMOS](https://github.com/llmos-ai/llmos-operator) cluster using the [Helm](https://helm.sh) package manager.

For more details, please refer to the following resources:

- [Snapshot CRDs](https://github.com/kubernetes-csi/external-snapshotter/tree/master/client/config/crd)
- [Snapshot Controller Documentation](https://github.com/kubernetes-csi/external-snapshotter/tree/master/deploy/kubernetes/snapshot-controller)