#!/bin/sh
set -o errexit

# Allow override of demo SQL file URL
NETBOX_SQL_DUMP_URL="${NETBOX_SQL_DUMP_URL:-https://raw.githubusercontent.com/netbox-community/netbox-demo-data/master/sql/netbox-demo-v4.1.sql}"

TMP_SQL_FILE=$(mktemp /tmp/netbox-data-dump.XXXXXXX.sql) || exit 1

# Download the SQL dump
curl -k "${NETBOX_SQL_DUMP_URL}" > "${TMP_SQL_FILE}"

# Load it into the database
psql "user=netbox host=netbox-db.${NAMESPACE}.svc.cluster.local" netbox -q -f "${TMP_SQL_FILE}"
rm "${TMP_SQL_FILE}"

# Load additional local data
psql "user=netbox host=netbox-db.${NAMESPACE}.svc.cluster.local" netbox -q -f /load-data-job/local-data-setup.sql
