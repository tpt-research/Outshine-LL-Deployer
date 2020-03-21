FROM golang:1.14 AS builder

WORKDIR /home/outshine/builder

COPY . .

ENV REDIS_IP "172.17.0.14"
ENV REDIS_IP "6379"

RUN go mod download
RUN go mod vendor

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o outshine-deployer

FROM scratch

ENV REDIS_IP "172.17.0.14"
ENV REDIS_IP "6379"

COPY --from=builder /home/outshine/builder/outshine-deployer .

EXPOSE 7137

CMD ["./outshine-deployer"]


