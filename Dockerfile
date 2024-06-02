FROM golang:alpine3.20 as builder

WORKDIR /app

COPY . .

RUN go mod tidy && go build -o url-shortener cmd/api/main.go

FROM scratch

COPY --from=builder /app/url-shortener /

EXPOSE 8080

CMD ["./url-shortener"]
