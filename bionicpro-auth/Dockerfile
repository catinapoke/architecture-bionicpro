FROM golang:1.26-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bionicpro-auth .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates
COPY --from=build /bionicpro-auth /bionicpro-auth

EXPOSE 8000
ENTRYPOINT ["/bionicpro-auth"]
