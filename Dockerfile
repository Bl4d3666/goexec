FROM golang:1.24-alpine AS goexec-builder
LABEL builder="true"

WORKDIR /go/src/

COPY cmd/ cmd/
COPY internal/ internal/
COPY pkg/ pkg/
COPY main.go go.mod go.sum ./

ENV CGO_ENABLED=0

RUN go mod download
RUN go build -ldflags="-s -w" -o /go/bin/goexec

# [For debugging]
#FROM alpine:3 AS goexec

FROM scratch AS goexec
COPY --from="goexec-builder" /go/bin/goexec /usr/local/bin/goexec

WORKDIR /io
ENTRYPOINT ["/usr/local/bin/goexec"]
