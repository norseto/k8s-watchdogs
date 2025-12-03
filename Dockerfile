FROM golang:1.24.11-alpine AS build

ARG GITVERSION
ARG MODULE_PACKAGE

RUN mkdir -p /build /dist
COPY . /build/
WORKDIR /build

ENV CGO_ENABLED=0
RUN go mod download \
	&& go vet cmd/watchdogs/*.go \
	&& CGO_ENABLED=0 go build -ldflags=all="-X ${MODULE_PACKAGE}.GitVersion=${GITVERSION}" -o /build/watchdogs cmd/watchdogs/*.go \
	&& cp watchdogs /dist \
	&& cp LICENSE /dist \
	;

FROM gcr.io/distroless/static-debian12
WORKDIR /
COPY --from=build --chown=65534:65534 /dist /
USER 65534
CMD ["/watchdogs"]
