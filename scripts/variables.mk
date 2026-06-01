GIT_TAG     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "local")
BUILD_TIME  := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -X main.version=$(GIT_TAG) \
           -X main.revision=$(GIT_COMMIT) \
           -X main.buildRFC3339=$(BUILD_TIME)
