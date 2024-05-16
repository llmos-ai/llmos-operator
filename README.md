# [WIP]LLMOS-Controller

LLMOS-Controller is a Kubernetes controller that helps to manage the lifecycle of [LLMOS](https://github.com/llmos-ai/llmos) and its system components.

## Description

## Getting Started

### Prerequisites
- Go version v1.22.0+
- Kubectl version v1.29.0+.
- Access to a Kubernetes v1.29.0+ cluster.
- Helm v3.0.0+

### Installation
To deploy the 1Block.AI on your k8s cluster, you can use the following commands:

**Install the CRDs into the cluster:**

```sh
$ make install-crd
```

**Deploy the llmos-controller to the cluster:**

```sh
$ make install
```

### Uninstall
**Delete the CRDs and llmos-controller from the cluster:**

```sh
$ make uninstall-crd && make uninstall
```

## License

Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

