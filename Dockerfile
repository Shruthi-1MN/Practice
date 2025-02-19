# Dockerfile
FROM golang:1.23-alpine as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o mathapi .

FROM alpine:latest

WORKDIR /root/
COPY --from=builder /app/mathapi .

ENV MONGODB_URI=mongodb://mongodb:27017
ENV JWT_SECRET=supersecret
ENV PORT=8080

EXPOSE 8080
CMD ["./mathapi"]