FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/ticket-service .

FROM alpine:3.20
COPY --from=build /out/ticket-service /ticket-service
CMD ["/ticket-service"]
