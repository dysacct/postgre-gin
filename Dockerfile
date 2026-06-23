FROM 1181.s.kuaicdn.cn:11818/docker.io/library/golang:1.26.3-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
ENV GOPROXY=https://goproxy.cn,direct
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o cmdb-api .

FROM 1181.s.kuaicdn.cn:11818/docker.io/library/alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/cmdb-api .
EXPOSE 34185
CMD ["./cmdb-api"]
