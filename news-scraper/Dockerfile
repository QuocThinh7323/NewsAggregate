FROM golang:1.23-alpine
# Removed invalid instruction

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o scraper .

CMD ["./news-scraper"]
