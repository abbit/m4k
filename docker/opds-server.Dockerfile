FROM golang:1.22.10

WORKDIR /app

COPY . .

RUN go build -o main cmd/opds_server/main.go

EXPOSE 6333

CMD ["./main"]
