FROM golang:1.20-alpine as BUILD

RUN mkdir -p /build /dist
COPY . /build/
WORKDIR /build

ENV CGO_ENABLED=0
RUN go install github.com/Songmu/gocredits/cmd/gocredits@latest \
	# && gocredits . > /dist/CREDITS \
	&& go mod download \
	&& go vet cmd/evicted-cleaner/*.go \
	&& CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /build/evicted-cleaner cmd/evicted-cleaner/*.go \
	&& cp evicted-cleaner /dist \
	&& cp LICENSE /dist \
	;

FROM scratch
WORKDIR /
COPY --from=BUILD /dist /
CMD ["/evicted-cleaner"]
