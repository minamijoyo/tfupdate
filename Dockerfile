FROM golang:1.13.3-alpine3.10 AS builder
RUN apk --no-cache add make git
WORKDIR /work

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN make build

FROM alpine:3.10
COPY --from=builder /work/bin/tfupdate /usr/local/bin/
ENTRYPOINT ["tfupdate"]
