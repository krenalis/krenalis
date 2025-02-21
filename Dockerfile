# Stage 0: Meergo Building Stage.

# Keep in sync with the version within ".github/workflows/main.yml".
# Keep in sync with the version within "go.mod".
FROM golang:1.24

WORKDIR /meergo
# Note that this command copies all files present in the local repository,
# including unversioned files, so a reproducible build can be achieved by
# checking out a new, freshly downloaded repository of Meergo.
COPY ./ ./
WORKDIR ./cmd/meergo
RUN go generate
WORKDIR ../../
RUN go build -tags osusergo,netgo -trimpath ./cmd/meergo

# Stage 1: Meergo Execution Stage.

# Since the Meergo build requires the Go toolchain, while its execution does
# not, a multi-stage build is used here to have, as the resulting image, an
# image that contains only the Meergo executable.
FROM scratch
COPY --from=0 /meergo/meergo /bin/meergo
WORKDIR /bin
ENTRYPOINT ["/bin/meergo"]
