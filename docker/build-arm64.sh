#!/usr/bin/env bash
#
# Build a native arm64 xrpld Docker image for Apple Silicon.
#
# Usage:
#   ./docker/build-arm64.sh                        # default image name
#   ./docker/build-arm64.sh my-xrpld:custom-tag    # custom image name
#
# The build takes 30-60+ minutes on the first run (compiling Boost, gRPC,
# wasmi, etc.). Subsequent builds use Docker layer caching.
#
# After building, set docker_image in bedrock.toml:
#   [local_node]
#   docker_image = "lejamon/rippled-smart-contracts-vault:arm64"

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

IMAGE_NAME="${1:-lejamon/rippled-smart-contracts-vault:arm64}"

echo "Building native arm64 xrpld image: ${IMAGE_NAME}"
echo "This will take 30-60+ minutes on the first build."
echo ""

# Verify we're on arm64
ARCH=$(uname -m)
if [ "$ARCH" != "arm64" ] && [ "$ARCH" != "aarch64" ]; then
    echo "Warning: Current architecture is ${ARCH}, not arm64."
    echo "The resulting image is intended for arm64/aarch64 systems."
    echo ""
fi

docker buildx build \
    --platform linux/arm64 \
    -f "${SCRIPT_DIR}/Dockerfile.arm64" \
    -t "${IMAGE_NAME}" \
    --load \
    "${PROJECT_ROOT}"

echo ""
echo "Build complete: ${IMAGE_NAME}"
echo ""
echo "Verify the image works:"
echo "  docker run --rm ${IMAGE_NAME} --version"
echo ""
echo "To use with Bedrock, add to your bedrock.toml:"
echo "  [local_node]"
echo "  docker_image = \"${IMAGE_NAME}\""
