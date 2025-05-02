FROM golang:1.24 as builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o promsql ./main.go

# --- Stage 2: Minimal runtime image ---
FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /app/promsql .

ENTRYPOINT ["./promsql"]