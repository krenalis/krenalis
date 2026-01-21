# Stage 0: Meergo Building Stage.

# Keep in sync with the version within ".github/workflows/go-run-test-commit.yml".
# Keep in sync with the version within ".github/workflows/send-sourcemaps-to-sentry.yml".
# Keep in sync with the version within "go.mod".
FROM golang:1.25-alpine3.23

WORKDIR /meergo

# Copy the Admin files.
RUN mkdir admin
COPY admin/*.go admin
COPY admin/package.json admin/package-lock.json admin/tsconfig.json admin
COPY admin/src admin/src
COPY admin/public admin/public/
COPY admin/node_modules_vendor admin/node_modules_vendor/
COPY admin/debugid admin/debugid/

# Copy the Go files.
COPY go.mod go.sum ./
COPY vendor vendor
COPY cmd cmd
COPY connectors connectors
COPY core core
COPY tools tools
COPY warehouses warehouses
COPY *.go ./

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

# Create a directory that can be mounted to contain the transformation
# functions.
#
# It's necessary to create it here, in the Dockerfile, for permissions reasons:
# otherwise, if it's created later (e.g., by Docker Compose), it will be created
# for the 'root' user, but since this container's user will be switched to
# 'meergouser', it won't have sufficient privileges to write to it. Therefore,
# we'll create it here, with the correct privileges already in place.
#
# This part should be simplified (i.e. removed) when we will implement
# https://github.com/meergo/meergo/issues/1962.
#
RUN mkdir -p /var/meergo/transformation-functions
RUN chown meergouser:meergouser /var/meergo/transformation-functions

# Create an user 'transformeruser' which will be used to run transformation
# functions executables.
RUN useradd transformeruser
RUN echo 'permit nopass meergouser as transformeruser' > /etc/doas.conf

USER meergouser
WORKDIR /home/meergouser
ENTRYPOINT ["/bin/meergo"]
