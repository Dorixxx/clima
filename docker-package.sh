#!/usr/bin/env bash

set -euo pipefail

IMAGE_NAME="${IMAGE_NAME:-cli-proxy-api}"
TAG="${TAG:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
VERSION="${VERSION:-${TAG}}"
COMMIT="${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo none)}"
BUILD_DATE="${BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
OUTPUT_DIR="${OUTPUT_DIR:-dist}"
SAFE_IMAGE_NAME="${IMAGE_NAME//\//_}"
ARCHIVE_PATH="${ARCHIVE_PATH:-${OUTPUT_DIR}/${SAFE_IMAGE_NAME}_${TAG}.tar}"
IMAGE_REF="${IMAGE_NAME}:${TAG}"

mkdir -p "${OUTPUT_DIR}"

echo "Building image: ${IMAGE_REF}"
echo "  Version: ${VERSION}"
echo "  Commit: ${COMMIT}"
echo "  Build Date: ${BUILD_DATE}"

docker build \
  --build-arg VERSION="${VERSION}" \
  --build-arg COMMIT="${COMMIT}" \
  --build-arg BUILD_DATE="${BUILD_DATE}" \
  -t "${IMAGE_REF}" \
  .

echo "Saving image archive: ${ARCHIVE_PATH}"
docker save -o "${ARCHIVE_PATH}" "${IMAGE_REF}"

echo
echo "Done."
echo "Local image: ${IMAGE_REF}"
echo "Archive: ${ARCHIVE_PATH}"
