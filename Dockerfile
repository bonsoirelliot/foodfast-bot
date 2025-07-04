FROM golang:1.23-alpine

WORKDIR /bot

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -v -x -o bot ./cmd/main.go

EXPOSE 8081

CMD ["./bot"]