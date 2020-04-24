# First step to build binary
FROM golang:1.14.2-alpine3.11 as build-env

# All these steps will be cached
RUN mkdir /work
WORKDIR /work
COPY go.mod .
COPY go.sum .

# Get dependancies - will also be cached if we won't change mod/sum
RUN go mod download

# COPY the source code as the last step
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o /go/bin/go-statsd-proxy

# Second step to build minimal image
FROM scratch

COPY --from=build-env /go/bin/go-statsd-proxy /bin/go-statsd-proxy

ENTRYPOINT ["/bin/go-statsd-proxy"]