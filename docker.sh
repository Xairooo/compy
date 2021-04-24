#!/bin/sh

if [ -n "${CERT_GENERATE}" ] || [ -r /opt/compy/ssl/cert.crt ] ; then
echo "Generating server certificate"
openssl req -x509 -newkey rsa:2048 -nodes -keyout cert.key -out cert.crt -days 3650 -subj "/CN=${CERT_DOMAIN}"
echo Done
CERT_FROM=Generated
else
CERT_FROM=Provided
fi

if [ -n "${CA_GENERATE}" ] || [ -r /opt/compy/ssl/ca.crt ] ; then
echo "Generating CA certificate"
openssl req -x509 -newkey rsa:2048 -nodes -keyout ca.key -out ca.crt -days 3650 -subj "/CN=${CA_DOMAIN}"
echo Done
CA_FROM=Generated
else
CA_FROM=Provided
fi

echo "${CERT_FROM} server certificate:"
cat ./ssl/cert.crt
echo
echo "${CA_FROM} CA certificate:"
cat ./ssl/ca.crt

exec ./compy \
    -cert ./ssl/cert.crt -key ./ssl/cert.key \
    -ca ./ssl/ca.crt -cakey ./ssl/ca.key \
    :9999
