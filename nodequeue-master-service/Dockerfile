FROM golang:1.21-alpine AS builder

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/master-service .

FROM gcr.io/distroless/static-debian12:nonroot

ENV PORT=8081
EXPOSE 8081

COPY --from=builder /out/master-service /master-service

USER nonroot:nonroot

ENTRYPOINT ["/master-service"]

