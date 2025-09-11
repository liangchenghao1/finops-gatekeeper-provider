#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
cd "${REPO_ROOT}" || exit 1
NAMESPACE=${NAMESPACE:-kube-system}

generate() {
    # generate CA key and certificate
    echo "Generating CA key and certificate for finops-gatekeeper-provider..."
    openssl genrsa -out ca.key 2048
    openssl req -new -x509 -days 365 -key ca.key -subj "/O=Gatekeeper/CN=Gatekeeper Root CA" -out ca.crt

    # generate server key and certificate
    echo "Generating server key and certificate for finops-gatekeeper-provider..."
    openssl genrsa -out tls.key 2048
    openssl req -newkey rsa:2048 -nodes -keyout tls.key -subj "/CN=finops-gatekeeper-provider.${NAMESPACE}" -out server.csr
    openssl x509 -req -extfile <(printf "subjectAltName=DNS:finops-gatekeeper-provider.%s" "${NAMESPACE}") -days 365 -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out tls.crt
}

mkdir -p "${REPO_ROOT}/certs"
pushd "${REPO_ROOT}/certs"
generate
popd