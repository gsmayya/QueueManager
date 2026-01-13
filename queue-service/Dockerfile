FROM golang:1.24-alpine AS builder

# Build from repo root context so we can include `queue-common` (used via go.mod replace).
WORKDIR /src

# Copy only module manifests first for better layer caching.
COPY queue-common/go.mod queue-common/go.mod
COPY queue-service/go.mod queue-service/go.sum queue-service/

WORKDIR /src/queue-service
RUN go mod download

# Copy full sources.
WORKDIR /src
COPY queue-common/ queue-common/
COPY queue-service/ queue-service/

# Build a small static binary.
WORKDIR /src/queue-service
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/queue-service .

FROM gcr.io/distroless/static-debian12:nonroot

ENV PORT=8080
EXPOSE 8080

COPY --from=builder /out/queue-service /queue-service

USER nonroot:nonroot

ENTRYPOINT ["/queue-service"]

