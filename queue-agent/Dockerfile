FROM golang:1.25-alpine AS builder

WORKDIR /src

# Copy only go.mod files first for better caching.
COPY queue-agent/go.mod ./queue-agent/go.mod
COPY queue-common/go.mod ./queue-common/go.mod

WORKDIR /src/queue-agent
RUN go mod download

# Now copy the sources.
WORKDIR /src
COPY queue-agent ./queue-agent
COPY queue-common ./queue-common

WORKDIR /src/queue-agent
ENV CGO_ENABLED=0
RUN go build -o /out/queue-agent .

FROM alpine:3.20 AS runner

RUN adduser -D -H -u 10001 appuser
WORKDIR /app
COPY --from=builder /out/queue-agent ./queue-agent

ENV QUEUE_AGENT_HTTP_ADDR=:8090
EXPOSE 8090

USER appuser
CMD ["./queue-agent"]

