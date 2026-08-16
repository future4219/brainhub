FROM golang:1.22-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/brainhub ./api \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/brainhub-migrate ./cmd/migrate

FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
    && addgroup -S brainhub \
    && adduser -S -G brainhub brainhub
COPY --from=build /out/brainhub /usr/local/bin/brainhub
COPY --from=build /out/brainhub-migrate /usr/local/bin/brainhub-migrate

USER brainhub
EXPOSE 8080
CMD ["sh", "-c", "brainhub-migrate up && exec brainhub"]
