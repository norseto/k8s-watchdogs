FROM golang:1.20-alpine as BUILD

RUN mkdir -p /build /dist
COPY . /build/
WORKDIR /build

ENV CGO_ENABLED=0
RUN go install github.com/Songmu/gocredits/cmd/gocredits@latest \
	&& go mod download \
	&& go vet cmd/pod-rebalancer/*.go \
	&& CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /build/pod-rebalancer cmd/pod-rebalancer/*.go \
	&& cp pod-rebalancer /dist \
	&& cp LICENSE /dist \
	;
#RUN go install github.com/Songmu/gocredits/cmd/gocredits@latest \
#	&& go mod download \
#	&& go vet cmd/pod-rebalancer/*.go \
#	&& CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /build/pod-rebalancer cmd/pod-rebalancer/*.go \
#	&& gocredits > /dist/CREDITS \
#	&& cp pod-rebalancer /dist \
#	&& cp LICENSE /dist \
#	;

FROM scratch
WORKDIR /
COPY --from=BUILD /dist /
CMD ["/pod-rebalancer"]
