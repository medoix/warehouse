FROM golang:alpine AS builder
WORKDIR /app
COPY . /app
# Genereate pkger.go file with static content to embed
RUN go get github.com/markbates/pkger/cmd/pkger && pkger
# Build warehouse binary
RUN CGO_ENABLED=0 go build -o warehouse .

FROM scratch
EXPOSE 8080
COPY --from=builder /app/warehouse .

ENTRYPOINT ["./warehouse", "-d", "/data"]
