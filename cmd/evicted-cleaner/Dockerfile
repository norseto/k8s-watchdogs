FROM golang:1.13-alpine as BUILD

RUN mkdir -p /build
RUN mkdir /dist
WORKDIR /build

ENV CGO_ENABLED=0
COPY . /build/
RUN go get github.com/Songmu/gocredits/cmd/gocredits \
	&& go mod download \
	&& go vet cmd/evicted-cleaner/*.go \
	&& CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /build/evicted-cleaner cmd/evicted-cleaner/*.go \
	&& gocredits > /dist/CREDITS \
	&& cp evicted-cleaner /dist \
	&& cp LICENSE /dist \
	;

FROM scratch
WORKDIR /
COPY --from=BUILD /dist /
CMD ["/evicted-cleaner"]
