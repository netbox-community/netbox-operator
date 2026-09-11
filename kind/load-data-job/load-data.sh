#!/bin/sh
set -o errexit

# Allow override of demo SQL file URL
NETBOX_SQL_DUMP_URL="${NETBOX_SQL_DUMP_URL:-https://raw.githubusercontent.com/netbox-community/netbox-demo-data/master/sql/netbox-demo-v4.1.sql}"

TMP_SQL_FILE=$(mktemp /tmp/netbox-data-dump.XXXXXXX.sql) || exit 1

# Download the SQL dump
curl -k "${NETBOX_SQL_DUMP_URL}" > "${TMP_SQL_FILE}"

# The dump is a plain pg_dump without DROP statements, so start from an empty schema
psql "user=netbox host=netbox-db.${NAMESPACE}.svc.cluster.local" netbox -q \
  -c 'DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;'

# Load it into the database
psql "user=netbox host=netbox-db.${NAMESPACE}.svc.cluster.local" netbox -q -f "${TMP_SQL_FILE}"
rm "${TMP_SQL_FILE}"

# Load additional local data
psql "user=netbox host=netbox-db.${NAMESPACE}.svc.cluster.local" netbox -q -f /load-data-job/local-data-setup.sql
