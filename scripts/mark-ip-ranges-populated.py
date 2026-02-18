# When updating to NetBox Version v4.3 or newer this script
# can be used to patch the mark-populated field of the IP ranges in Netbox
# which contain '// managed by netbox-operator' in the description
# (v4.2 NetBox versions are not compatible with the IP range claim controller)

import argparse
import os
import pynetbox
import requests
from pprint import pprint

NETBOX_API = os.getenv("NETBOX_API", "http://netbox")
TOKEN = os.getenv("NETBOX_TOKEN", "0123456789abcdef0123456789abcdef01234567")
CA_CERT = os.getenv("CA_CERT")
MANAGED_MARKER = "// managed by netbox-operator"


def main():
    parser = argparse.ArgumentParser(
        description="List IP ranges managed by netbox-operator and optionally mark them as populated."
    )
    parser.add_argument(
        "--mark-populated",
        action="store_true",
        help="If set, patch the 'mark_populated' custom field of matching IP ranges to true.",
    )
    parser.add_argument(
        "--limit",
        type=int,
        default=None,
        help="Maximum number of IP ranges to process (for pagination). If not set, all matches are processed.",
    )
    parser.add_argument(
        "--offset",
        type=int,
        default=0,
        help="Number of matching IP ranges to skip before processing (for pagination). Default: 0.",
    )
    parser.add_argument(
        "--not-populated-only",
        action="store_true",
        help="If set, only show/patch IP ranges where mark_populated is false.",
    )
    args = parser.parse_args()

    try:
        nb = pynetbox.api(
            NETBOX_API,
            token=TOKEN,
        )

        # Configure SSL verification if CA cert is provided
        if CA_CERT:
            session = requests.Session()
            session.verify = CA_CERT
            nb.http_session = session
            print(f"Using CA certificate: {CA_CERT}")

    except pynetbox.RequestError as e:
        pprint(e.error)
        raise SystemExit(f"Failed to connect to NetBox at {NETBOX_API}")

    print(f"Connected to NetBox API at {NETBOX_API}")

    api_limit = args.limit or 1000
    ip_ranges = nb.ipam.ip_ranges.filter(limit=api_limit, offset=args.offset)

    managed_ranges = []
    for ip_range in ip_ranges:
        description = ip_range.description or ""
        if MANAGED_MARKER not in description:
            continue

        if args.not_populated_only:
            if getattr(ip_range, "mark_populated", False):
                continue

        prefix_part = description.split("//")[0].strip()
        managed_ranges.append((ip_range, prefix_part))

    if not managed_ranges:
        print("No matching IP ranges found.")
        return

    total = len(managed_ranges)

    if total == 1000:
        print("Warning - pagination limit reached, use the offset flag to query more IP ranges")

    print(f"Showing {len(managed_ranges)} of {total} matching IP range(s) (offset={args.offset}, limit={args.limit or '1000'}):\n")

    for ip_range, prefix_part in managed_ranges:
        print(f"[id={ip_range.id}] {ip_range.start_address} - {ip_range.end_address}: {prefix_part}  (mark_populated={ip_range.mark_populated})")

    if args.mark_populated:
        print(f"\nMarking {len(managed_ranges)} IP range(s) as populated...")
        for ip_range, _ in managed_ranges:
            try:
                ip_range.mark_populated = True
                ip_range.save()
                print(f"  ✓ Patched IP range {ip_range.id}")
            except pynetbox.RequestError as e:
                pprint(e.error)
                print(f"  ✗ Failed to patch IP range {ip_range.id}")

        print("Done.")


if __name__ == "__main__":
    main()
