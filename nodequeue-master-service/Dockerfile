FROM golang:1.23-alpine AS builder

WORKDIR /src

# Copy only module manifests first for better layer caching.
COPY queue-common/go.mod queue-common/go.mod
COPY nodequeue-master-service/go.mod nodequeue-master-service/go.sum nodequeue-master-service/

WORKDIR /src/nodequeue-master-service
RUN go mod download

# Copy full sources.
WORKDIR /src
COPY queue-common/ queue-common/
COPY nodequeue-master-service/ nodequeue-master-service/

# Build a small static binary.
WORKDIR /src/nodequeue-master-service

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/master-service .

FROM gcr.io/distroless/static-debian12:nonroot

ENV PORT=8081
EXPOSE 8081

COPY --from=builder /out/master-service /master-service

USER nonroot:nonroot

ENTRYPOINT ["/master-service"]

