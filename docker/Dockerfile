FROM golang:1.17-alpine3.15 as builder

# Install tools and snappy lib
RUN apk update && apk add --no-cache --virtual .build-deps \
        g++ \
        gcc \
        make \
        git \
        openssl \
        snappy-dev \
        leveldb-dev \
        ca-certificates

WORKDIR /ndidplatform/migration-tools
COPY . /ndidplatform/migration-tools/

ENV CGO_ENABLED=1
ENV CGO_LDFLAGS="-lsnappy"

WORKDIR /ndidplatform/migration-tools
RUN go build \
    -tags "cleveldb" \
    -o ./build/migration-tools


FROM alpine:3.15
LABEL maintainer="NDID IT Team <it@ndid.co.th>"

# Set umask to 027
RUN umask 027 && echo "umask 0027" >> /etc/profile

COPY --from=builder /var/cache/apk /var/cache/apk

# Install snappy lib used by LevelDB.
# Install bash shell for convenience.
RUN apk add --no-cache \
      bash \
      leveldb \
      snappy \
      tzdata && \
    rm -rf /var/cache/apk

COPY --from=builder /ndidplatform/migration-tools/build/migration-tools /usr/bin/migration-tools

ENTRYPOINT [ "migration-tools" ]
