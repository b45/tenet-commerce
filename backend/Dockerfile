FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/tenet-api ./cmd/api

FROM alpine:3.22
RUN addgroup -S app && adduser -S -G app app && apk add --no-cache ca-certificates wget
COPY --from=build /out/tenet-api /tenet-api
EXPOSE 8081
USER app:app
ENTRYPOINT ["/tenet-api"]
