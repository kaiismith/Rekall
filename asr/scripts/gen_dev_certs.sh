#!/usr/bin/env bash
# Generates a local CA + server cert + client cert for mTLS development.
#
# Output: tls/{ca.crt,ca.key,server.crt,server.key,client.crt,client.key}
#
# DO NOT use these certs in production. The CA key is unencrypted on disk.
set -euo pipefail

OUT="${1:-$(cd "$(dirname "$0")/.." && pwd)/tls}"
mkdir -p "$OUT"
cd "$OUT"

if [[ -f ca.crt ]]; then
    echo "tls/ already populated; skipping (delete the directory to regenerate)"
    exit 0
fi

# CA
openssl genrsa -out ca.key 4096
openssl req -x509 -new -nodes -key ca.key -sha256 -days 3650 \
    -subj "/CN=rekall-asr-dev-ca" -out ca.crt

# Server
openssl genrsa -out server.key 2048
openssl req -new -key server.key -subj "/CN=rekall-asr" -out server.csr
cat > server.ext <<EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage=digitalSignature,keyEncipherment
extendedKeyUsage=serverAuth
subjectAltName=DNS:rekall-asr,DNS:asr,DNS:localhost,IP:127.0.0.1
EOF
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
    -out server.crt -days 825 -sha256 -extfile server.ext
rm server.csr server.ext

# Client (Rekall Go backend)
openssl genrsa -out client.key 2048
openssl req -new -key client.key -subj "/CN=rekall-backend" -out client.csr
cat > client.ext <<EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage=digitalSignature,keyEncipherment
extendedKeyUsage=clientAuth
EOF
openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
    -out client.crt -days 825 -sha256 -extfile client.ext
rm client.csr client.ext

chmod 600 *.key
echo "wrote dev certs to $OUT"
