FROM golang:1.24-alpine AS builder

# Build from repo root context so we can include `queue-common` (used via go.mod replace).
WORKDIR /src

# Copy only module manifests first for better layer caching.
COPY queue-common/go.mod queue-common/go.mod
COPY nodequeue-service/go.mod nodequeue-service/go.sum nodequeue-service/

WORKDIR /src/nodequeue-service
RUN go mod download

# Copy full sources.
WORKDIR /src
COPY queue-common/ queue-common/
COPY nodequeue-service/ nodequeue-service/

# Build a small static binary.
WORKDIR /src/nodequeue-service
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/nodequeue-service .

FROM gcr.io/distroless/static-debian12:nonroot

ENV PORT=8080
EXPOSE 8080

COPY --from=builder /out/nodequeue-service /nodequeue-service

USER nonroot:nonroot

ENTRYPOINT ["/nodequeue-service"]

