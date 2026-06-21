FROM golang:1.25 AS build-env

ARG COMMIT_SHA=dev

ENV GO111MODULE=on    \
    GOTOOLCHAIN=auto  \
    CGO_ENABLED=0     \
    GOOS=linux        \
    GOARCH=amd64

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY cmd/project-00 project-00/
COPY internal/ internal/
COPY pkg/ pkg/
COPY VERSION ./
RUN VERSION=$(cat VERSION) && \
    go build -o app -ldflags "-X main.AppVersion=${VERSION}-${COMMIT_SHA}" project-00/*.go


FROM alpine

WORKDIR /go/bin
RUN apk add --no-cache ca-certificates
COPY application.yml /go/bin/application.yml
COPY --from=build-env /build/app .
RUN chmod +x app

ENTRYPOINT ["/go/bin/app"]
