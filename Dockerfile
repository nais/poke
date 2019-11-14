FROM golang:1.13-alpine as builder
RUN apk add --no-cache git
ENV GOOS=linux
ENV CGO_ENABLED=0
ENV GO111MODULE=on
COPY . /src
WORKDIR /src
RUN rm -f go.sum
RUN go get
RUN go test .
RUN go build -a -installsuffix cgo -o poke

FROM alpine:3.10
MAINTAINER Johnny Horvi <johnny.horvi@nav.no>
WORKDIR /app
COPY --from=builder /src/poke /app/poke
ENTRYPOINT ["/app/poke"]
