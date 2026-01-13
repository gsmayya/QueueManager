FROM golang:1.24-alpine AS builder

WORKDIR /src

# Copy only module manifests first for better layer caching.
COPY queue-common/go.mod queue-common/go.mod
COPY queue-admin/go.mod queue-admin/go.sum queue-admin/

WORKDIR /src/queue-admin
RUN go mod download

# Copy full sources.
WORKDIR /src
COPY queue-common/ queue-common/
COPY queue-admin/ queue-admin/

# Build a small static binary.
WORKDIR /src/queue-admin

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/queue-admin .

FROM gcr.io/distroless/static-debian12:nonroot

ENV PORT=8081
EXPOSE 8081

COPY --from=builder /out/queue-admin /queue-admin

USER nonroot:nonroot

ENTRYPOINT ["/queue-admin"]

