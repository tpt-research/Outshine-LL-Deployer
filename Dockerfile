FROM golang:1.14

WORKDIR /home/outshine/builder

COPY . .

ENV REDIS_IP="172.17.0.14"
ENV REDIS_PORT="6379"
ENV APIKEY="admin:nimda"

RUN go mod download
RUN go mod vendor

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o outshine-deployer

CMD ["./outshine-deployer"]




