#!/bin/sh
set -e

ADMIN_API="http://garage:3903/v2"
HEADER="Authorization: Bearer ${GARAGE_ADMIN_TOKEN}"
CONFIG_FILE="/app/config.json"

echo "Waiting for Garage Admin API..."
until curl -s -f -H "$HEADER" "$ADMIN_API/GetClusterStatus" >/dev/null 2>&1; do
  sleep 2
done

echo "Configuring cluster layout..."
# Fetch the auto-generated node ID dynamically
NODE_ID=$(curl -s -H "$HEADER" "$ADMIN_API/GetClusterStatus" | jq -r '.nodes[0].id')

echo "configuring node with id: ${NODE_ID}"

LAYOUT_PAYLOAD=$(jq -n \
  --arg id "$NODE_ID" \
  '{
    "parameters": null,
    "roles": [
      {
        "id": $id,
        "zone": "dc1",
        "capacity": 10737418240,
        "tags": []
      }
    ]
  }')


#echo "$LAYOUT_PAYLOAD"
# 1. Assign layout (Zone: dc1, Capacity: 10GB)
curl -s -X POST "$ADMIN_API/UpdateClusterLayout" \
  -H "$HEADER" \
  -H "Content-Type: application/json" \
  -d "${LAYOUT_PAYLOAD}" > /dev/null


# 2. Apply layout version 1
VERSION=$(curl -s -H "$HEADER" "$ADMIN_API/GetClusterLayout" | jq '.version')

echo "Current layout version: $VERSION!"

if [[ $VERSION -eq 0 ]]; then
    curl -s -X POST "$ADMIN_API/ApplyClusterLayout" \
    -H "$HEADER" \
    -H "Content-Type: application/json" \
    -d '{"version": 1}' >/dev/null
fi

echo "Creating buckets and access keys..."

# Create app access key
KEY_NAME=$(jq -r '.key' "$CONFIG_FILE")
KEY_EXISTS=$(curl -s -H "$HEADER" "$ADMIN_API/ListKeys" | jq -r '.[] | .name' | grep -F "$KEY_NAME" || true)

if [[ -z "$KEY_EXISTS" ]]; then
   KEY_PAYLOAD=$(jq -n \
  --arg name "$KEY_NAME" \
  --arg id "$CUSTOM_ACCESS_KEY_ID" \
  --arg secret "$CUSTOM_SECRET_KEY" \
  '{
    "accessKeyId": $id,
    "name": $name,
    "secretAccessKey": $secret
  }')

    echo $KEY_PAYLOAD

    curl -s -X POST "$ADMIN_API/ImportKey" \
      -H "$HEADER" \
      -H "Content-Type: application/json" \
      -d "${KEY_PAYLOAD}" > /dev/null
else
    echo "Key '$KEY_NAME' already exists."
fi

# Create bucket for game assets
echo "Provisioning buckets..."
for bucket in $(jq -r '.buckets[]' "$CONFIG_FILE"); do
  if [[ -z "$bucket" ]]; then
    continue
  fi

  EXISTING=$(curl -s -H "$HEADER" "$ADMIN_API/ListBuckets" | jq -r '.[] | .globalAliases[]?' | grep -F "$bucket" || true)
  if [ -z "$EXISTING" ]; then
    curl -s -X POST "$ADMIN_API/CreateBucket" \
      -H "$HEADER" \
      -H "Content-Type: application/json" \
      -d "$(jq -n --arg b "$bucket" '{globalAlias: $b}')" >/dev/null
    echo "Created bucket: $bucket"
  else
    echo "Bucket '$bucket' already exists."
  fi
done

BUCKET_IDS=$(curl -s -H "$HEADER" "$ADMIN_API/ListBuckets" | jq -r '.[] | .id')

for BUCKET_ID in $BUCKET_IDS; do
    ALLOW_KEY_PAYLOAD=$(jq -n \
  --arg key "$CUSTOM_ACCESS_KEY_ID" --arg  bucket "$BUCKET_ID" \
   '{
    "accessKeyId": $key,
    "bucketId": $bucket,
    "permissions": {
        "owner": true,
        "read": true,
        "write": true
                }
    }')
    curl -s -X POST "$ADMIN_API/AllowBucketKey" \
      -H "$HEADER" \
      -H "Content-Type: application/json" \
      -d "${ALLOW_KEY_PAYLOAD}" > /dev/null
done


echo "Garage fully configured via Admin API v2!"