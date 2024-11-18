FROM golang:1.23-alpine AS build

WORKDIR /build

RUN apk update && apk add --no-cache musl-dev gcc build-base

COPY go.mod ./
COPY go.sum ./
RUN go mod download
RUN go mod verify

COPY . .

RUN go build

FROM alpine

COPY --from=build /build/sendibot /usr/bin/sendibot

ENTRYPOINT [ "/usr/bin/sendibot" ]
