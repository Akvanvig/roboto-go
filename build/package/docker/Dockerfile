FROM golang:bullseye AS roboto-builder

# dependencies required for layeh.com/gopus 
# RUN apk add --update opus-dev gcc libgcc libstdc++ g++
RUN apt-get update \
  && apt-get install -y libopus-dev gcc libgcc-10-dev libstdc++-10-dev g++

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
ARG TARGETPLATFORM
# Download latest yt-dlp for aarch64
COPY ./tools/dockerplatform_to_yt-dlp_file.sh /tools/script.sh
RUN apk add --no-cache wget
RUN YT_DLP=$(/bin/sh /tools/script.sh ${TARGETPLATFORM}) && wget -O /out/yt-dlp https://github.com/yt-dlp/yt-dlp/releases/latest/download/${YT_DLP}
RUN chmod 755 /out/yt-dlp

# copy files to final container
#FROM gcr.io/distroless/static-debian11
FROM debian:bullseye-slim

RUN apt update \
  && apt-get install -y ffmpeg ca-certificates \
  && apt-get clean
USER nobody:nogroup
WORKDIR  /bot/

COPY --from=roboto-builder /out/roboto /bot/roboto
COPY --from=roboto-downloader /out/yt-dlp /bot/yt-dlp
COPY ./assets /bot/assets/

ENTRYPOINT [ "/bot/roboto" ]
