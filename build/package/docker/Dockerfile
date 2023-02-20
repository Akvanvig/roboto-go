FROM golang:alpine AS roboto-builder

# dependencies required for layeh.com/gopus 
RUN apk add --update opus-dev gcc libgcc libstdc++ g++

# copy git data to build container
COPY ./build /data/build/
COPY ./cmd /data/cmd/
COPY ./internal /data/internal/
COPY ./go.mod ./go.sum /data/
WORKDIR  /data/

# build binary
RUN go get -d -v ./...
RUN CGO_ENABLED=1 go build -ldflags="-w -s" -o /out/ ./...

FROM alpine:3 AS roboto-downloader
# Download latest yt-dlp for aarch64
ADD https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux_aarch64 /out/yt-dlp
RUN chmod +x /bot/yt-dlp

# copy files to final container
#FROM gcr.io/distroless/static-debian11
FROM alpine:3

RUN apk add --no-cache ffmpeg
USER nobody:nogroup

COPY --from=roboto-builder /out/roboto /bot/roboto
COPY --from=roboto-downloader /out/yt-dlp /bot/yt-dlp
COPY ./assets /bot/assets/

ENTRYPOINT [ "/bot/roboto" ]
