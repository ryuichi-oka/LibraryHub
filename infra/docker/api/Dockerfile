FROM golang:1.24

WORKDIR /app

COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download

COPY apps/api/ .

EXPOSE 8080

CMD ["go", "run", "./cmd/server"]
