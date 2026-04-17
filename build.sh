#!/bin/bash
# GaiaSec NodeAgent Cross-Platform Build Script
# This script builds the NodeAgent for multiple platforms

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Version information
BUILD_VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")

echo -e "${BLUE}Building GaiaSec NodeAgent for multiple platforms...${NC}"
echo -e "${YELLOW}Version: ${BUILD_VERSION}${NC}"
echo ""

# Ensure dependencies are downloaded
echo -e "${BLUE}Downloading dependencies...${NC}"
go mod tidy
echo -e "${GREEN}✓ Dependencies updated${NC}"
echo ""

# Clean build flag
if [ "$1" == "--clean" ]; then
    echo -e "${YELLOW}Cleaning build cache...${NC}"
    go clean -cache -modcache 2>/dev/null || true
fi

# Output directory (absolute path)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUTPUT_DIR="${SCRIPT_DIR}/agent/"
mkdir -p "$OUTPUT_DIR"

echo -e "${BLUE}Output directory: ${OUTPUT_DIR}${NC}"
echo ""

# Build for different platforms
# Format: "OS/ARCH[/<CROSS_COMPILER>]"
PLATFORMS=(
    "linux/amd64"
    "linux/arm64/aarch64-linux-gnu-gcc"
)

TOTAL=${#PLATFORMS[@]}
CURRENT=0

    for PLATFORM in "${PLATFORMS[@]}"; do
    CURRENT=$((CURRENT + 1))

    CC=""
    CC_OPT=""
    IFS='/' read -r OS ARCH CC <<< "$PLATFORM"

    echo -e "${BLUE}[${CURRENT}/${TOTAL}] Building for $OS/$ARCH...${NC}"

    # Set output filename with version info
    OUTPUT_NAME="nodeagent-${OS}-${ARCH}"

    if [ -n "$CC" ]; then
        CC_OPT="CC=$CC"
        echo -e "${YELLOW}  Using cross-compiler: $CC${NC}"
    fi

    # Build the binary with stable version info only so identical sources
    # produce identical ELF outputs across repeated builds.
    env GOOS=$OS GOARCH=$ARCH CGO_ENABLED=1 $CC_OPT go build \
        -ldflags="-w -s -X 'gaiasec-nodeagent/pkg/version.Version=${BUILD_VERSION}'" \
        -trimpath \
        -o "$OUTPUT_DIR/$OUTPUT_NAME" \
        ./cmd/nodeagent

    if [ $? -eq 0 ]; then
        # Show file size
        SIZE=$(ls -lh "$OUTPUT_DIR/$OUTPUT_NAME" | awk '{print $5}')
        echo -e "${GREEN}✓ Successfully built ${OUTPUT_NAME} (Size: ${SIZE})${NC}"
    else
        echo -e "${RED}✗ Failed to build ${OUTPUT_NAME}${NC}"
        exit 1
    fi
done

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Build completed successfully!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "${BLUE}Output directory:${NC} ${OUTPUT_DIR}"
echo -e "${BLUE}Generated binaries:${NC}"
ls -lh "$OUTPUT_DIR/" | grep -E "^-" | awk '{print "  " $9 " (" $5 ")"}'
echo ""
echo -e "${YELLOW}Build Summary:${NC}"
echo -e "${YELLOW}  Version: ${BUILD_VERSION}${NC}"
echo -e "${YELLOW}  Platforms: ${TOTAL}${NC}"
echo ""
chmod a+rx ./sync.sh
./sync.sh
echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Sync completed successfully!${NC}"
echo -e "${GREEN}========================================${NC}"
