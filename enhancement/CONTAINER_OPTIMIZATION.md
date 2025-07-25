# Container Image Size Optimization Guide

This document outlines the container image optimizations implemented for the LLMOS Operator project.

## Overview

The optimization strategy focuses on:
- **Multi-stage builds** with Chainguard Wolfi base images
- **Essential runtime tools** (bash, tini, curl, kubectl, jq)
- **Optimized Go build flags**
- **Enhanced Docker BuildKit caching**
- **Reduced build context size**

## Optimizations Implemented

### 1. Dockerfiles Optimization

#### Main Operator (`package/Dockerfile`)
- **Before**: OpenSUSE Leap 15.6 base (~200MB+)
- **After**: Chainguard Wolfi base with essential tools (~60MB)
- **Features**:
  - Multi-stage build with UI asset pre-download
  - Essential runtime tools (bash, tini, curl, kubectl, jq)
  - Unified entrypoint script for all modes

#### Webhook & Downloader (`package/Dockerfile-webhook`, `package/Dockerfile-downloader`)
- **Before**: OpenSUSE Leap 15.6 with unnecessary packages
- **After**: Chainguard Wolfi base with essential tools
- **Size reduction**: ~80% smaller

### 2. Go Build Optimizations

Updated build flags in `.goreleaser.yaml`:
```yaml
ldflags:
  - -s          # Strip symbol table
  - -w          # Strip debug info
  - -buildid=   # Remove build ID for reproducible builds
flags:
  - -trimpath   # Remove absolute paths
tags:
  - netgo       # Pure Go networking
  - osusergo    # Pure Go user/group lookups
```

### 3. Docker BuildKit Enhancements

Added to all Docker builds:
```yaml
build_flag_templates:
  - "--cache-from=type=gha"           # GitHub Actions cache
  - "--cache-to=type=gha,mode=max"    # Aggressive caching
  - "--build-arg=BUILDKIT_INLINE_CACHE=1"  # Inline cache
```

### 4. Enhanced .dockerignore

Excludes unnecessary files:
- Documentation and samples
- Development tools and configs
- CI/CD files
- Build artifacts
- IDE files

## Expected Results

| Component | Before | After | Reduction |
|-----------|--------|-------|-----------|
| Main Operator | ~350MB | ~60MB | 83% |
| Webhook | ~300MB | ~30MB | 90% |
| Downloader | ~300MB | ~30MB | 90% |

## Benefits

1. **Faster deployments**: Smaller images pull faster
2. **Reduced storage costs**: Less registry storage needed
3. **Better security**: Minimal attack surface
4. **Improved caching**: Better layer reuse
5. **Faster CI/CD**: Reduced build and push times

## Usage

### Building locally
```bash
# Build with goreleaser
goreleaser build --snapshot --clean

# Build Docker images
goreleaser release --snapshot --clean
```

### CI/CD Integration

The optimizations are automatically applied when using goreleaser in CI/CD pipelines. Ensure:
- Docker BuildKit is enabled
- GitHub Actions cache is available (for GHA)
- Proper registry credentials are configured

## Monitoring

### Image Size Verification
```bash
# Check image sizes
docker images | grep llmos-operator

# Detailed image analysis
docker history <image-name>
```

### Build Performance
- Monitor build times in CI/CD
- Check cache hit rates
- Verify layer reuse efficiency

## Maintenance

### Regular Updates
1. **Base images**: Keep Chainguard Wolfi images updated
2. **Go version**: Update golang:1.24-alpine in UI downloader stage
3. **Dependencies**: Monitor for security updates
4. **Tools**: Update essential tools (kubectl, jq) as needed

### Troubleshooting

#### Common Issues
1. **Signal handling**: Go runtime handles signals properly without init system
2. **UI assets not found**: Verify download URLs and versions
3. **Permission issues**: Check file permissions in distroless
4. **Wrong mode**: Verify LLMOS_MODE environment variable is set correctly

#### Debug Commands
```bash
# Inspect image layers
docker history --no-trunc <image>

# Check file permissions
docker run --rm -it <image> ls -la /usr/bin/

# Verify UI assets
docker run --rm -it <image> ls -la /usr/share/llmos/
```

## Security Considerations

- **Chainguard Wolfi**: Minimal, security-focused base image
- **Essential tools only**: bash, tini, curl, kubectl, jq for operational needs
- **Regular updates**: Chainguard provides timely security updates
- **Minimal dependencies**: Reduced attack surface
- **No secrets**: Build-time secrets properly handled

### 5. Entrypoint Consolidation

Consolidated three separate entrypoint scripts into one unified script:
- **Before**: `entrypoint.sh`, `entrypoint-webhook.sh`, `entrypoint-downloader.sh`
- **After**: Single `entrypoint.sh` with mode detection
- **Benefits**: Easier maintenance, consistent behavior, reduced complexity

#### Usage
```bash
# Via environment variable
LLMOS_MODE=webhook ./entrypoint.sh

# Via command argument
./entrypoint.sh webhook

# Default mode (apiserver)
./entrypoint.sh
```

## Future Optimizations

1. **UPX compression**: Consider binary compression
2. **Scratch base**: For even smaller webhook/downloader images
3. **Multi-arch optimization**: Architecture-specific optimizations
4. **Layer optimization**: Further reduce layer count

---

*Last updated: $(date)*
*For questions or improvements, please open an issue or PR.*