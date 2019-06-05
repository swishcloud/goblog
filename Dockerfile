FROM golang:1.8



WORKDIR /workspace/go/app

COPY . /workspace/go/app



RUN go install



CMD ["goblog"]
