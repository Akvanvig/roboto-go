FROM golang:1.23 AS builder

COPY ./go.mod ./go.sum /build/
COPY ./cmd /build/cmd/
COPY ./internal /build/internal/
WORKDIR /build

RUN go mod download
RUN CGO_ENABLED=0 go build -o /out/roboto ./cmd/roboto

FROM gcr.io/distroless/static-debian12

COPY --from=builder /out/roboto /bot/roboto
ENTRYPOINT [ "/bot/roboto" ]