# Keep the Go toolchain native while cross-compiling for each target platform.
FROM --platform=$BUILDPLATFORM golang:1.26.6-alpine AS build

RUN mkdir -p /build /dist
WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download
COPY . /build/

ARG GITVERSION
ARG MODULE_PACKAGE
ARG TARGETOS
ARG TARGETARCH

ENV CGO_ENABLED=0
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go vet cmd/watchdogs/*.go \
	&& GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags=all="-X ${MODULE_PACKAGE}.GitVersion=${GITVERSION}" -o /build/watchdogs cmd/watchdogs/*.go \
	&& cp watchdogs /dist \
	&& cp LICENSE /dist \
	;

FROM gcr.io/distroless/static-debian12
WORKDIR /
COPY --from=build --chown=65534:65534 /dist /
USER 65534
CMD ["/watchdogs"]
