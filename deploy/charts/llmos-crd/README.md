# LLMOS-CRD
This is the CRD chart for LLMOS-Operator. Using seperated chart for CRD installation because the total size of the chart is too big.

## Notes
If the operator or webhook deployment is list/watch any new third-party resources, you will also need to add those CRDs to the `templates`,
otherwise the deployment will fail to start because those CRDs are not exist.


