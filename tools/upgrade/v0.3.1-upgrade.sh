#!/bin/bash -xe

info() {
    echo '[INFO] ' "$@"
}

fatal() {
    echo '[ERROR] ' "$@" >&2
    exit 1
}

warning() {
    echo '[WARNING] ' "$@"
}

# Check if node's label nvidia.com/cuda.driver-version.major version is >= 575
check_nvidia_driver_version() {
    info "Checking NVIDIA CUDA driver version..."
    
    # Get all nodes with the nvidia.com/cuda.driver-version.major label
    NODES_WITH_NVIDIA=$(kubectl get nodes -l "nvidia.com/cuda.driver-version.major" -o jsonpath='{.items[*].metadata.name}' 2>/dev/null || true)
    
    if [ -z "$NODES_WITH_NVIDIA" ]; then
        info "No nodes found with nvidia.com/cuda.driver-version.major label, skipping NVIDIA driver version check"
        return 0
    fi
    
    for node in $NODES_WITH_NVIDIA; do
        DRIVER_VERSION=$(kubectl get node "$node" -o jsonpath='{.metadata.labels.nvidia\.com/cuda\.driver-version\.major}' 2>/dev/null || true)
        
        if [ -n "$DRIVER_VERSION" ]; then
            info "Node $node has NVIDIA CUDA driver version: $DRIVER_VERSION"
            
            if [ "$DRIVER_VERSION" -lt 575 ]; then
                fatal "Node $node has NVIDIA CUDA driver version $DRIVER_VERSION which is less than required version 575, please update the CUDA driver first"
            fi
            
            info "Node $node NVIDIA CUDA driver version $DRIVER_VERSION meets requirement (>= 575)"
        fi
    done
    
    info "All nodes with NVIDIA CUDA drivers meet the version requirement"
}

# Delete specific CRDs with version v1alpha1 if they exist
delete_v1alpha1_crds() {
    info "Checking and deleting v1alpha1 CRDs if they exist..."
    
    CRDS_TO_DELETE=(
        "volumegroupsnapshotclasses.groupsnapshot.storage.k8s.io"
        "volumegroupsnapshotcontents.groupsnapshot.storage.k8s.io"
        "volumegroupsnapshots.groupsnapshot.storage.k8s.io"
    )
    
    for crd in "${CRDS_TO_DELETE[@]}"; do
        info "Checking CRD: $crd"
        
        # Check if CRD exists and has v1alpha1 version
        CRD_EXISTS=$(kubectl get crd "$crd" -o jsonpath='{.metadata.name}' 2>/dev/null || true)
        
        if [ -n "$CRD_EXISTS" ]; then
            # Check if it has v1alpha1 version
            HAS_V1ALPHA1=$(kubectl get crd "$crd" -o jsonpath='{.spec.versions[?(@.name=="v1alpha1")].name}' 2>/dev/null || true)
            
            if [ -n "$HAS_V1ALPHA1" ]; then
                info "Deleting CRD $crd with v1alpha1 version"
                kubectl delete crd "$crd" --ignore-not-found=true
                info "Successfully deleted CRD: $crd"
            else
                info "CRD $crd exists but does not have v1alpha1 version, skipping"
            fi
        else
            info "CRD $crd does not exist, skipping"
        fi
    done
    
    info "Finished checking and deleting v1alpha1 CRDs"
}

# Apply upgrade configuration
apply_upgrade_config() {
    info "Applying v0.3.1 upgrade configuration..."
    
    # Check and remove existing upgrade object if it exists
    info "Checking for existing upgrade object..."
    EXISTING_UPGRADE=$(kubectl get upgrades.management.llmos.ai upgrade-v030-v1 -o jsonpath='{.metadata.name}' 2>/dev/null || true)
    
    if [ -n "$EXISTING_UPGRADE" ]; then
        info "Found existing upgrade object: $EXISTING_UPGRADE, deleting it..."
        kubectl delete upgrades.management.llmos.ai upgrade-v030-v1 --ignore-not-found=true
        info "Successfully deleted existing upgrade object"
    else
        info "No existing upgrade object found, proceeding with creation"
    fi
    
    # Create temporary file with upgrade configuration
    UPGRADE_CONFIG=$(mktemp)
    
    cat > "$UPGRADE_CONFIG" << 'EOF'
apiVersion: management.llmos.ai/v1
kind: Version
metadata:
  name: v0.3.1
spec:
  minUpgradableVersion: v0.2.0
  kubernetesVersion: v1.33.1+k3s1
  releaseDate: "2025-08-25"
  tags: ["stable"]
---
apiVersion: management.llmos.ai/v1
kind: Upgrade
metadata:
  name: upgrade-v030-v1
spec:
  version: v0.3.1 # The version to upgrade to
  kubernetesVersion: v1.33.1+k3s1
  registry: "ghcr.io/llmos-ai"
EOF
    
    info "Applying upgrade configuration with kubectl"
    kubectl apply -f "$UPGRADE_CONFIG"
    
    # Clean up temporary file
    rm -f "$UPGRADE_CONFIG"
    
    info "Successfully applied v0.3.1 upgrade configuration"
}

# Main execution
main() {
    info "Starting v0.3.1 upgrade process..."
    
    # Step 1: Check NVIDIA CUDA driver version
    check_nvidia_driver_version
    
    # Step 2: Delete v1alpha1 CRDs
    delete_v1alpha1_crds
    
    # Step 3: Apply upgrade configuration
    apply_upgrade_config
    
    info "v0.3.1 upgrade process completed successfully"
}

# Execute main function
main "$@"