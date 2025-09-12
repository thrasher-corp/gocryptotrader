FROM golang:1.25

# Install GCC with multi-architecture support (needed for SQLite library)
RUN apt-get update && apt-get install -y gcc-multilib && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . /app

CMD ["go", "test", "./..."]