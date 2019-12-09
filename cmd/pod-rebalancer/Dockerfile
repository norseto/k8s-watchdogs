FROM golang:1.13-alpine as BUILD

RUN mkdir -p /build
RUN mkdir /dist
WORKDIR /build

ENV CGO_ENABLED=0
COPY . /build/
RUN go get github.com/Songmu/gocredits/cmd/gocredits \
	&& go mod download \
	&& go vet cmd/pod-rebalancer/*.go \
	&& CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /build/pod-rebalancer cmd/pod-rebalancer/*.go \
	&& gocredits > /dist/CREDITS \
	&& cp pod-rebalancer /dist \
	&& cp LICENSE /dist \
	;

FROM scratch
WORKDIR /
COPY --from=BUILD /dist /
CMD ["/pod-rebalancer"]
