FROM ubuntu:22.04 AS compy-builder
MAINTAINER Barna Csorogi <barnacs@justletit.be>

WORKDIR /compy
COPY . .
RUN DEBIAN_FRONTEND=noninteractive apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get upgrade -y && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y \
        golang-go \
        curl \
        g++ \
        git \
        libjpeg9 \
        libjpeg9-dev && \
    go mod init github.com/Xairooo/compy && \
    go mod tidy && \
    go build -v -o compy

FROM ubuntu:22.04
MAINTAINER Barna Csorogi <barnacs@justletit.be>

RUN DEBIAN_FRONTEND=noninteractive apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get upgrade -y && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y \
        libjpeg9 \
        libjpeg9-dev \
        openssl \
        ssl-cert && \
    DEBIAN_FRONTEND=noninteractive apt-get clean && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /opt/compy
COPY --from=compy-builder /compy/compy .
COPY ./docker.sh .
    
# TODO: configure HTTP BASIC authentication
# TODO: --user-provided certificates-- Solved
ENV \
    CERTIFICATE_DOMAIN="localhost"

VOLUME ["/opt/compy/ssl"]
EXPOSE 9999
ENTRYPOINT ["./docker.sh"]
