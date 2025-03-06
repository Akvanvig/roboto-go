FROM golang:1.23 AS builder

COPY ./cmd /build/cmd/
COPY ./internal /build/internal/
COPY ./go.mod ./go.sum /build/
WORKDIR /build

RUN go mod download
RUN CGO_ENABLED=0 go build -o /out/roboto ./cmd/roboto

FROM gcr.io/distroless/static-debian12

COPY --from=builder /out/roboto /bot/roboto
ENTRYPOINT [ "/bot/roboto" ]