#!/usr/bin/env bash
# ============================================================
# init-ssl.sh — Generate self-signed TLS certificate
# ============================================================
# Usage:
#   ./init-ssl.sh                   # certs saved to ./ssl/
#   ./init-ssl.sh /path/to/ssl      # custom output directory
#   DOMAIN=myhost.local ./init-ssl.sh
#
# This creates a self-signed certificate (valid 10 years) for
# development and internal deployment. For production, replace
# the generated files with real CA-signed certificates or use
# Let's Encrypt / ACME.
# ============================================================

set -euo pipefail

SSL_DIR="${1:-./ssl}"
DOMAIN="${DOMAIN:-cloud-agent.local}"
DAYS=3650  # 10 years

mkdir -p "$SSL_DIR"

# Skip generation if certs already exist (don't overwrite)
if [[ -f "$SSL_DIR/server.crt" && -f "$SSL_DIR/server.key" ]]; then
  echo "[init-ssl] Certificates already exist in $SSL_DIR — skipping."
  echo "           Delete them first if you want to regenerate."
  exit 0
fi

echo "[init-ssl] Generating self-signed TLS certificate ..."
echo "           Domain : $DOMAIN"
echo "           Output : $SSL_DIR/"
echo "           Validity: $DAYS days"

openssl req -x509 -nodes -newkey rsa:2048 \
  -days "$DAYS" \
  -keyout "$SSL_DIR/server.key" \
  -out    "$SSL_DIR/server.crt" \
  -subj   "/C=CN/ST=Beijing/L=Beijing/O=CloudAgent/OU=Dev/CN=$DOMAIN" \
  -addext "subjectAltName=DNS:$DOMAIN,DNS:localhost,IP:127.0.0.1" \
  2>/dev/null

chmod 600 "$SSL_DIR/server.key"
chmod 644 "$SSL_DIR/server.crt"

echo "[init-ssl] Done. Files created:"
echo "           $SSL_DIR/server.crt  (certificate)"
echo "           $SSL_DIR/server.key  (private key)"
echo ""
echo "  For production, replace these with CA-signed certificates."
echo "  To use Let's Encrypt, consider certbot or acme.sh."
