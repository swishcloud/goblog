#/bin/sh
if [ -d ./.dist ]
then
rm ./.dist -r
mkdir ./.dist
fi

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./.dist/app
cp templates  ./.dist -r
cp static  ./.dist -r
cp migrations  ./.dist -r
#sudo docker build --tag $IMAGE_TAG -f docker/dockerfile ./.dist