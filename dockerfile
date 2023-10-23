FROM golang:1.21-alpine AS build

RUN apk update && apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o app

FROM alpine:3.14

RUN apk --no-cache add ca-certificates

COPY --from=build /app/app .

CMD ["./app"]