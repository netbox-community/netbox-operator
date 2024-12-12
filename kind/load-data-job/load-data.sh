#!/bin/sh
set -o errexit

TMP_SQL_FILE=$(mktemp /tmp/netbox-data-dump.XXXXXXX.sql) || exit 1
curl -k https://raw.githubusercontent.com/netbox-community/netbox-demo-data/master/sql/netbox-demo-v4.1.sql > "${TMP_SQL_FILE}"
psql "user=netbox host=netbox-db.${NAMESPACE}.svc.cluster.local" netbox -q -f "${TMP_SQL_FILE}"
rm "${TMP_SQL_FILE}"
psql "user=netbox host=netbox-db.${NAMESPACE}.svc.cluster.local" netbox -q -f /load-data-job/local-demo-data.sql
