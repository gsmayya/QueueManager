FROM golang:1.21-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build a small static binary.
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/nodequeue-service .

FROM gcr.io/distroless/static-debian12:nonroot

ENV PORT=8080
EXPOSE 8080

COPY --from=builder /out/nodequeue-service /nodequeue-service

USER nonroot:nonroot

ENTRYPOINT ["/nodequeue-service"]

