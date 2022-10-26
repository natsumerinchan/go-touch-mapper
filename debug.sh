echo 'building...'
go build -ldflags="-s -w" -o go-touch-mapper
echo 'build success!'
sudo ./go-touch-mapper -a -r -d -c configs/smash_legends.json