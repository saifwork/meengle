FROM  golang:1.23.2-alpine3.19

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o socket-service .

EXPOSE 8080

CMD [ "./socket-service" ]
