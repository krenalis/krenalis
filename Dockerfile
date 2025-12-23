# Stage 0: Meergo Building Stage.

# Keep in sync with the version within ".github/workflows/go-run-test-commit.yml".
# Keep in sync with the version within ".github/workflows/send-sourcemaps-to-sentry.yml".
# Keep in sync with the version within "go.mod".

# TODO: valutare come questo impatta anche altri punti con la versione di Go, e i commenti
FROM golang:1.25-alpine3.23

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
RUN go generate
RUN go build -tags osusergo,netgo -trimpath

# Stage 1: Meergo Execution Stage.

# Since the Meergo build requires the Go toolchain, while its execution does
# not, a multi-stage build is used here to have, as the resulting image, an
# image that contains only the Meergo executable and the Python and JavaScript
# (node) interpreters, for the transformation functions.
FROM alpine:3.23

# Install Python and Node.js.
RUN apk add --no-cache python3
RUN apk add --no-cache nodejs

# Copy the Meergo executable from stage 0 to stage 1.
COPY --from=0 /meergo/meergo /bin/meergo

# Install two packages:
#
#    doas    ->   provides the 'doas' command
#    shadow  ->   provides the 'useradd' command
RUN apk add doas shadow

# Create the user 'meergouser' (and its home directory): this will be used to
# run Meergo.
RUN useradd meergouser -m

# Create an user 'transformeruser' which will be used to run transformation
# functions executables.
RUN useradd transformeruser
RUN echo 'permit nopass meergouser as transformeruser' > /etc/doas.conf

USER meergouser
WORKDIR /home/meergouser
ENTRYPOINT ["/bin/meergo"]
