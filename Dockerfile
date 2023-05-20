FROM golang:1-alpine3.17 AS builder

RUN apk add --no-cache git ca-certificates build-base olm-dev
COPY . /build
WORKDIR /build
RUN export MAUTRIX_VERSION=$(cat go.mod | grep 'maunium.net/go/mautrix ' | head -n1 | awk '{ print $2 }'); \
    go build -ldflags "-X main.Tag=$(git describe --exact-match --tags 2>/dev/null) -X main.Commit=$(git rev-parse HEAD) -X 'main.BuildTime=`date '+%b %_d %Y, %H:%M:%S'`' -X 'maunium.net/go/mautrix.GoModVersion=$MAUTRIX_VERSION'" -o /usr/bin/botbot

FROM alpine:3.17

RUN apk add --no-cache ca-certificates olm bash
COPY --from=builder /usr/bin/botbot /usr/bin/botbot

CMD ["/usr/bin/botbot"]
