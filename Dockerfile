# Build on the native platform and cross-compile for the target: Go needs no
# emulation to do that, so multi-platform builds stay fast.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

ARG VERSION=dev
ARG REVISION=unknown
ARG BUILD_DATE=unknown

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build \
    -ldflags "-X main.Version=${VERSION} -X main.Revision=${REVISION} -X main.BuildDate=${BUILD_DATE}" \
    -o miniflux-mcp .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/miniflux-mcp .

EXPOSE 8080

CMD ["./miniflux-mcp"]
