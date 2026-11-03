FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /stockledger ./cmd/stockledger && CGO_ENABLED=0 go build -o /outbox-worker ./cmd/outbox-worker
FROM gcr.io/distroless/static-debian12
COPY --from=build /stockledger /stockledger
COPY --from=build /outbox-worker /outbox-worker
USER nonroot:nonroot
