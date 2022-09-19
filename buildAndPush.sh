CGO_ENABLE=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o go-touch-mapper
adb push ./go-touch-mapper /data/local/tmp