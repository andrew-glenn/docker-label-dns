# syntax=docker/dockerfile:1
FROM golang:1.19-alpine
WORKDIR /app
COPY . .
RUN go build -o /container-dns-from-labels
CMD [ "/container-dns-from-labels", "server", "--port",  "5555"]
