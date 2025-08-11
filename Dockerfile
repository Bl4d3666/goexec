FROM golang:1.24-alpine AS goexec-builder
LABEL builder="true"

WORKDIR /go/src/github.com/github.com/FalconOpsLLC/goexec

COPY . .

ENV CGO_ENABLED=0

RUN go mod download
RUN go build -ldflags="-s -w" -o /go/bin/goexec

FROM scratch
COPY --from="goexec-builder" /go/bin/goexec /goexec

WORKDIR /io
VOLUME ["/io"]
ENTRYPOINT ["/goexec"]
