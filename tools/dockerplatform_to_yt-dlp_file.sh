if [ "$#" -ne 1 ]; then
    echo "USAGE $0 <DOCKER BUILDPLATFORM>" >&2
    exit 1
fi

case $1 in
  linux/amd64)
    echo "yt-dlp_linux"
    ;;

  linux/arm/v7)
    echo "yt-dlp_linux_armv7l"
    ;;

  linux/arm64)
    echo "yt-dlp_linux_aarch64"
    ;;

  *)
    echo "$1 IS AN UNSUPPORTED PLATFORM" >&2
    exit 1
    ;;
esac

exit 0