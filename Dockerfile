FROM golang:1.8

WORKDIR /go/src/app

COPY . /go/src/app

RUN go version

RUN git clone https://github.com/udhos/update-golang

RUN update-golang/update-golang.sh

RUN go get github.com/github-123456/gostudy/aesencryption

RUN go get github.com/github-123456/goweb

RUN go get github.com/go-sql-driver/mysql

RUN go install

CMD goblog