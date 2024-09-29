# System-Chart
System-Charts is a collection of Helm Charts that used by the LLMOS system services.

## Update Dependency charts
- Run `make helm-dep-system-charts` to update system-charts dependencies if you have changed the version or modified the dependency charts.
- Update the dependency [CRDs](../llmos-crd/templates) of the related system-charts, this is used by the llmos-operator to bootstrap API server correctly with listing/watching those CRDs.
- Bump the [go.mod](https://github.com/llmos-ai/llmos-operator/blob/main/go.mod) dependency version & regenerate the controller types & code via `make generate` if applicable.

## Build & Package Charts
- Run `make build-system-charts` to build the system-charts locally.
- Run `make package-system-charts-repo` to package the system-charts repo image.
  - Set `export UPLOAD_CHARTS=false` to skip uploading the charts to the S3 repo.

## Deploy Charts locally
To Test the system-charts locally, run the following command:
```shell
docker run -d -it --name repo -p 8088:80 ghcr.io/llmos-ai/system-charts-repo:main-head
```
And then you can access the repo at `http://localhost:8088`.

