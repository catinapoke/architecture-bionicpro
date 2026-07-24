#!/bin/sh
set -eu

CONNECT_URL="${CONNECT_URL:-http://connect:8083}"
CONNECTOR_FILE="${CONNECTOR_FILE:-/debezium/crm-connector.json}"
CONNECTOR_NAME="${CONNECTOR_NAME:-crm-connector}"

until curl -fsS "${CONNECT_URL}/connectors" >/dev/null; do
  echo "Waiting for Kafka Connect at ${CONNECT_URL}..."
  sleep 2
done

if curl -fsS "${CONNECT_URL}/connectors/${CONNECTOR_NAME}" >/dev/null 2>&1; then
  echo "Connector ${CONNECTOR_NAME} already exists"
  exit 0
fi

curl -fsS \
  -X POST \
  -H "Accept: application/json" \
  -H "Content-Type: application/json" \
  --data-binary "@${CONNECTOR_FILE}" \
  "${CONNECT_URL}/connectors"

echo "Registered ${CONNECTOR_NAME}"
