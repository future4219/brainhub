FROM golang:1.22-bookworm AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/brainhub-shim ./shim

FROM gbrain

COPY --from=build --chown=bun:bun /out/brainhub-shim /usr/local/bin/brainhub-shim
WORKDIR /brains
USER bun
CMD ["brainhub-shim"]
