# tfupdate
FROM golang:1.21-alpine3.18 AS tfupdate
RUN apk --no-cache add make git
WORKDIR /work

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN make build

# hub
# The linux binary for hub can not run on alpine.
# So we need to build it from source.
# https://github.com/github/hub/issues/1818
FROM golang:1.21-alpine3.18 AS hub
RUN apk add --no-cache bash git
RUN git clone https://github.com/github/hub /work
WORKDIR /work
RUN ./script/build -o bin/hub

# runtime
# Note: Required Tools for Primary Containers on CircleCI
# https://circleci.com/docs/2.0/custom-images/#required-tools-for-primary-containers
FROM alpine:3.18
RUN apk --no-cache add bash git openssh-client tar gzip ca-certificates jq openssl curl
COPY --from=tfupdate /work/bin/tfupdate /usr/local/bin/
COPY --from=hub /work/bin/hub /usr/local/bin/
ENTRYPOINT ["tfupdate"]
