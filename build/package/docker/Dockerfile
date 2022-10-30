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

# clone yt-dlp release branch
# https://github.com/yt-dlp/yt-dlp/tree/release
FROM bitnami/git AS yt-dlp-git
WORKDIR /out/
RUN git clone https://github.com/yt-dlp/yt-dlp.git --single-branch -b release .


# build yt-dlp
FROM python:3.11 AS yt-dlp-builder
COPY --from=yt-dlp-git /out /data/
WORKDIR /data/

RUN python3 -m pip install -U pyinstaller -r requirements.txt
RUN python3 pyinst.py
RUN mkdir /out && cp dist/yt-dlp_linux* /out/yt-dlp

# copy files to final container
#FROM gcr.io/distroless/static-debian11
FROM alpine:3

RUN apk add --no-cache ffmpeg
COPY --from=yt-dlp-builder /out/yt-dlp /bot/yt-dlp
COPY --from=roboto-builder /out/roboto /bot/roboto
COPY ./assets /bot/assets/

USER nobody:nogroup

ENTRYPOINT [ "/bot/roboto" ]