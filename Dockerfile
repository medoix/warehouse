FROM golang:alpine AS builder
WORKDIR /app
COPY . /app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o warehouse .

FROM scratch
EXPOSE 8080
COPY --from=builder /app/warehouse .

CMD ["./warehouse"]
