FROM --platform=$BUILDPLATFORM golang:alpine AS roboto-builder

# copy git data to build container
COPY ./build /data/build/
COPY ./cmd /data/cmd/
COPY ./internal /data/internal/
COPY ./go.mod ./go.sum /data/
WORKDIR  /data/

# build binary
RUN apk add git
RUN go get -d -v ./...
RUN CGO_ENABLED=1 go build -ldflags="-w -s" -o /out/ ./...

# clone yt-dlp release branch
# https://github.com/yt-dlp/yt-dlp/tree/release
FROM alpine/git AS yt-dlp-git
WORKDIR /out/
RUN git clone https://github.com/yt-dlp/yt-dlp.git --single-branch -b release .


# build yt-dlp
FROM --platform=$BUILDPLATFORM python:3.10 AS yt-dlp-builder

COPY --from=yt-dlp-git /out /data/
WORKDIR /data/

RUN python3 -m pip install -U pyinstaller -r requirements.txt
#RUN python3 devscripts/make_lazy_extractors.py 
RUN python3 pyinst.py
RUN mkdir /out && cp dist/yt-dlp_linux /out/yt-dlp

# fetch ffmpeg
FROM --platform=$BUILDPLATFORM busybox AS ffmpeg-builder
ARG TARGETPLATFORM
WORKDIR /data
RUN \
    case ${TARGETPLATFORM} in \
        "linux/amd64") DOWNLOAD_URL="https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-amd64-static.tar.xz" ;; \
        "linux/arm64") DOWNLOAD_URL="https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-arm64-static.tar.xz" ;; \
        "linux/arm/v7") DOWNLOAD_URL="https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-armhf-static.tar.xz" ;; \
        "linux/armv6") DOWNLOAD_URL="https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-armel-static.tar.xz" ;; \
    esac && \
    wget $DOWNLOAD_URL -O ffmpeg.tar.xz &&\
    tar xvf ffmpeg.tar.xz && \
    mkdir /out && \
    cp ffmpeg*/ffmpeg /out/
    
# copy files to final container
FROM --platform=$BUILDPLATFORM gcr.io/distroless/static-debian11

COPY --from=roboto-builder /out/roboto /bot/roboto
COPY --from=yt-dlp-builder /out/yt-dlp /bot/yt-dlp
COPY --from=ffmpeg-builder /out/ffmpeg /bot/ffmpeg
COPY ./assets /bot/

USER nobody:nogroup

ENTRYPOINT [ "/bot/roboto" ]