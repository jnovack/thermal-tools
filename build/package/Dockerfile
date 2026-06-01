FROM golang:1.24-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG VERSION=dev
ARG REVISION=local
ARG BUILD_TIME=1970-01-01T00:00:00Z
RUN go build \
    -ldflags "-X main.version=${VERSION} -X main.revision=${REVISION} -X main.buildRFC3339=${BUILD_TIME}" \
    -o /bin/thermal-tools \
    ./cmd/thermal-tools

FROM scratch
COPY --from=builder /bin/thermal-tools /thermal-tools
ENTRYPOINT ["/thermal-tools"]
