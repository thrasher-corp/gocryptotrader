FROM golang:1.24-alpine

# Install GCC and musl-dev (needed for SQLite library)
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . /app

CMD ["go", "test", "./..."]