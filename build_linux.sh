cd ./yog
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build  -o ../out/EyeServer
cd ../eye
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build  -o ../out/YogEye
