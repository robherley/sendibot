FROM golang:1.23 AS build

WORKDIR /build

COPY go.mod ./
COPY go.sum ./
RUN go mod download
RUN go mod verify

COPY . .

RUN go build

FROM ubuntu:22.04

COPY --from=build /build/sendibot /usr/bin/sendibot

ENTRYPOINT [ "/usr/bin/sendibot" ]
