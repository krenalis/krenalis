# Stage 0: Meergo Building Stage.

# Keep in sync with the version within ".github/workflows/main.yml".
# Keep in sync with the version within ".github/workflows/send-sourcemaps-to-sentry.yml".
# Keep in sync with the version within "go.mod".
FROM golang:1.25

WORKDIR /meergo

# Pre-copy/cache go.mod for pre-downloading dependencies and only re-downloading
# them in subsequent builds if they change.
#
# Adapted from https://hub.docker.com/_/golang.
COPY go.mod go.sum ./
RUN go mod download -x

# Note that this command copies all files present in the local repository,
# including unversioned files, so a reproducible build can be achieved by
# checking out a new, freshly downloaded repository of Meergo.
COPY ./ ./
WORKDIR ./cmd/meergo
# Create a Go source file that imports the Filesystem connector, since this
# connector is not included by default in the Meergo command in the repository.
RUN printf 'package main\nimport _ "github.com/meergo/meergo/connectors/filesystem"\n' | tee import_filesystem.go
RUN go generate
WORKDIR ../../
RUN go build -tags osusergo,netgo -trimpath ./cmd/meergo

# Stage 1: Meergo Execution Stage.

# Since the Meergo build requires the Go toolchain, while its execution does
# not, a multi-stage build is used here to have, as the resulting image, an
# image that contains only the Meergo executable and the Python and JavaScript
# (node) interpreters, for the transformation functions.
FROM alpine:latest
RUN apk add --no-cache python3
RUN apk add --no-cache nodejs
COPY --from=0 /meergo/meergo /bin/meergo
WORKDIR /bin
ENTRYPOINT ["/bin/meergo"]
