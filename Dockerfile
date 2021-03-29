# Build Stage
FROM golang:1.12 as build

ENV GO111MODULE=on
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN go install ./cmd/mealplanner

EXPOSE 3000
ENTRYPOINT ["mealplanner"]
