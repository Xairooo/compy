FROM golang:latest as compy-builder
MAINTAINER Barna Csorogi <barnacs@justletit.be>

RUN DEBIAN_FRONTEND=noninteractive apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get upgrade -y && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y \
        curl \
        g++ \
        git \
        libjpeg-dev

#RUN mkdir -p /usr/local/ && \
#    curl -O https://storage.googleapis.com/golang/go1.9.linux-amd64.tar.gz && \
#    tar xf go1.9.linux-amd64.tar.gz -C /usr/local

#RUN mkdir -p /go/src/github.com/barnacs/compy/
#COPY . /go/src/github.com/barnacs/compy/
#WORKDIR /go/src/github.com/barnacs/compy
#RUN /usr/local/go/bin/go get -d -v ./...
#RUN /usr/local/go/bin/go build -v

RUN go get github.com/barnacs/compy
WORKDIR /go/src/github.com/barnacs/compy
#RUN go install

FROM ubuntu:20.04
MAINTAINER Barna Csorogi <barnacs@justletit.be>

RUN DEBIAN_FRONTEND=noninteractive apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get upgrade -y && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y \
        libjpeg8 \
        openssl \
        ssl-cert && \
    DEBIAN_FRONTEND=noninteractive apt-get clean && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /opt/compy
COPY \
    --from=compy-builder \
    /go/bin/compy \
    /opt/compy/
COPY \
    ./docker.sh \
    /opt/compy/
    
# TODO: configure HTTP BASIC authentication
# TODO: --user-provided certificates-- Solved
ENV \
    CERTIFICATE_DOMAIN="localhost"

VOLUME ["/opt/compy/ssl"]
EXPOSE 9999
ENTRYPOINT ["./docker.sh"]
