FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux go build -o portfolio .

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /app/portfolio /portfolio
EXPOSE 8080
ENTRYPOINT ["/portfolio"]
