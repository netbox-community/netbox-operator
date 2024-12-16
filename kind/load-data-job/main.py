import pynetbox
from pprint import pprint
from dataclasses import dataclass
import sys

nb = pynetbox.api(
    'http://localhost:8080',
    token='0123456789abcdef0123456789abcdef01234567'
)

# insert Tenants

@dataclass
class Tenant:
    name: str
    slug: str
    custom_fields: dict

tenants = [
    Tenant(
        name="MY_TENANT",
        slug="my_tenant",
        custom_fields={
            "cust_id": None,
        },
    ),
]

for tenant in tenants:
    try:
        nb.tenancy.tenants.create(
            name=tenant.name,
            slug=tenant.slug,
            custom_fields=tenant.custom_fields,
        )
    except pynetbox.RequestError as e:
        pprint(e.error)
        sys.exit(1)

# devices = list(nb.tenancy.tenants.all())
# for device in devices:
#     pprint(device)

# insert IPs

# insert Prefixes

"""
-- insert Prefix
-- 2.0.0.0/16
INSERT INTO public.ipam_prefix (created, last_updated, custom_field_data, prefix, status, is_pool, description, role_id, site_id, tenant_id, vlan_id, vrf_id, _children, _depth, mark_utilized, comments)
VALUES ('2024-06-14 10:01:10.094083+00', '2024-06-14 10:01:10.094095+00', '{}', '2.0.0.0/16', 'active', false, '', NULL, NULL, 100, NULL, NULL, 0, 0, false, '');

-- 2.1.0.0/24
INSERT INTO public.ipam_prefix (created, last_updated, custom_field_data, prefix, status, is_pool, description, role_id, site_id, tenant_id, vlan_id, vrf_id, _children, _depth, mark_utilized, comments)
VALUES ('2024-06-14 10:01:10.094083+00', '2024-06-14 10:01:10.094095+00', '{}', '2.1.0.0/24', 'active', false, '', NULL, NULL, 100, NULL, NULL, 0, 0, false, '');

-- 2.2.0.0/24
INSERT INTO public.ipam_prefix (created, last_updated, custom_field_data, prefix, status, is_pool, description, role_id, site_id, tenant_id, vlan_id, vrf_id, _children, _depth, mark_utilized, comments)
VALUES ('2024-06-14 10:01:10.094083+00', '2024-06-14 10:01:10.094095+00', '{}', '2.2.0.0/24', 'active', false, '', NULL, NULL, 100, NULL, NULL, 0, 0, false, '');

-- 3.0.0.0/24 - 3.0.8.0/24 (watch out for the upper/lower-case)
-- Pool 1, Production (IPv4)
INSERT INTO public.ipam_prefix (created, last_updated, custom_field_data, prefix, status, is_pool, description, role_id, site_id, tenant_id, vlan_id, vrf_id, _children, _depth, mark_utilized, comments)
VALUES ('2024-06-14 10:01:10.094083+00', '2024-06-14 10:01:10.094095+00', '{"environment": "Production", "poolName": "Pool 1", "cfDataTypeBool": true, "cfDataTypeInteger": 1}', '3.0.0.0/24', 'active', false, '', NULL, NULL, 100, NULL, NULL, 0, 0, false, '');

INSERT INTO public.ipam_prefix (created, last_updated, custom_field_data, prefix, status, is_pool, description, role_id, site_id, tenant_id, vlan_id, vrf_id, _children, _depth, mark_utilized, comments)
VALUES ('2024-06-14 10:01:10.094083+00', '2024-06-14 10:01:10.094095+00', '{"environment": "Production", "poolName": "Pool 1", "cfDataTypeBool": true, "cfDataTypeInteger": 1}', '3.0.1.0/24', 'active', false, '', NULL, 5, 100, NULL, NULL, 0, 0, false, '');

-- Pool 1, Development (IPv4)
INSERT INTO public.ipam_prefix (created, last_updated, custom_field_data, prefix, status, is_pool, description, role_id, site_id, tenant_id, vlan_id, vrf_id, _children, _depth, mark_utilized, comments)
VALUES ('2024-06-14 10:01:10.094083+00', '2024-06-14 10:01:10.094095+00', '{"environment": "Development", "poolName": "Pool 1", "cfDataTypeBool": false, "cfDataTypeInteger": 2}', '3.0.2.0/24', 'active', false, '', NULL, NULL, 100, NULL, NULL, 0, 0, false, '');

-- Pool 2, Production (IPv4)
INSERT INTO public.ipam_prefix (created, last_updated, custom_field_data, prefix, status, is_pool, description, role_id, site_id, tenant_id, vlan_id, vrf_id, _children, _depth, mark_utilized, comments)
VALUES ('2024-06-14 10:01:10.094083+00', '2024-06-14 10:01:10.094095+00', '{"environment": "Production", "poolName": "Pool 2", "cfDataTypeBool": true, "cfDataTypeInteger": 3}', '3.0.3.0/24', 'active', false, '', NULL, NULL, 100, NULL, NULL, 0, 0, false, '');

INSERT INTO public.ipam_prefix (created, last_updated, custom_field_data, prefix, status, is_pool, description, role_id, site_id, tenant_id, vlan_id, vrf_id, _children, _depth, mark_utilized, comments)
VALUES ('2024-06-14 10:01:10.094083+00', '2024-06-14 10:01:10.094095+00', '{"environment": "Production", "poolName": "Pool 2", "cfDataTypeBool": true, "cfDataTypeInteger": 3}', '3.0.4.0/24', 'active', false, '', NULL, 5, 100, NULL, NULL, 0, 0, false, '');

-- Pool 2, Development (IPv4)
INSERT INTO public.ipam_prefix (created, last_updated, custom_field_data, prefix, status, is_pool, description, role_id, site_id, tenant_id, vlan_id, vrf_id, _children, _depth, mark_utilized, comments)
VALUES ('2024-06-14 10:01:10.094083+00', '2024-06-14 10:01:10.094095+00', '{"environment": "Development", "poolName": "Pool 2", "cfDataTypeBool": false, "cfDataTypeInteger": 4}', '3.0.5.0/24', 'active', false, '', NULL, NULL, 100, NULL, NULL, 0, 0, false, '');

-- pool 3, production (IPv4)
INSERT INTO public.ipam_prefix (created, last_updated, custom_field_data, prefix, status, is_pool, description, role_id, site_id, tenant_id, vlan_id, vrf_id, _children, _depth, mark_utilized, comments)
VALUES ('2024-06-14 10:01:10.094083+00', '2024-06-14 10:01:10.094095+00', '{"environment": "production", "poolName": "pool 3", "cfDataTypeBool": true, "cfDataTypeInteger": 5}', '3.0.6.0/24', 'active', false, '', NULL, NULL, 100, NULL, NULL, 0, 0, false, '');

INSERT INTO public.ipam_prefix (created, last_updated, custom_field_data, prefix, status, is_pool, description, role_id, site_id, tenant_id, vlan_id, vrf_id, _children, _depth, mark_utilized, comments)
VALUES ('2024-06-14 10:01:10.094083+00', '2024-06-14 10:01:10.094095+00', '{"environment": "production", "poolName": "pool 3", "cfDataTypeBool": true, "cfDataTypeInteger": 5}', '3.0.7.0/24', 'active', false, '', NULL, 5, 100, NULL, NULL, 0, 0, false, '');

-- pool 3, development (IPv4)
INSERT INTO public.ipam_prefix (created, last_updated, custom_field_data, prefix, status, is_pool, description, role_id, site_id, tenant_id, vlan_id, vrf_id, _children, _depth, mark_utilized, comments)
VALUES ('2024-06-14 10:01:10.094083+00', '2024-06-14 10:01:10.094095+00', '{"environment": "development", "poolName": "pool 3", "cfDataTypeBool": false, "cfDataTypeInteger": 6}', '3.0.8.0/24', 'active', false, '', NULL, NULL, 100, NULL, NULL, 0, 0, false, '');

-- pool 4, production (IPv6)
INSERT INTO public.ipam_prefix (created, last_updated, custom_field_data, prefix, status, is_pool, description, role_id, site_id, tenant_id, vlan_id, vrf_id, _children, _depth, mark_utilized, comments)
VALUES ('2024-06-14 10:01:10.094083+00', '2024-06-14 10:01:10.094095+00', '{"environment": "production", "poolName": "pool 4", "cfDataTypeBool": true, "cfDataTypeInteger": 7}', '2::/64', 'active', false, '', NULL, NULL, 5, NULL, NULL, 0, 0, false, '');

INSERT INTO public.ipam_prefix (created, last_updated, custom_field_data, prefix, status, is_pool, description, role_id, site_id, tenant_id, vlan_id, vrf_id, _children, _depth, mark_utilized, comments)
VALUES ('2024-06-14 10:01:10.094083+00', '2024-06-14 10:01:10.094095+00', '{"environment": "production", "poolName": "pool 4", "cfDataTypeBool": true, "cfDataTypeInteger": 7}', '2:0:0:1::/64', 'active', false, '', NULL, 5, 5, NULL, NULL, 0, 0, false, '');

-- pool 4, development (IPv6)
INSERT INTO public.ipam_prefix (created, last_updated, custom_field_data, prefix, status, is_pool, description, role_id, site_id, tenant_id, vlan_id, vrf_id, _children, _depth, mark_utilized, comments)
VALUES ('2024-06-14 10:01:10.094083+00', '2024-06-14 10:01:10.094095+00', '{"environment": "development", "poolName": "pool 4", "cfDataTypeBool": false, "cfDataTypeInteger": 8}', '2:0:0:2::/64', 'active', false, '', NULL, NULL, 5, NULL, NULL, 0, 0, false, '');
"""

devices = list(nb.ipam.prefixes.all())
for device in devices:
    pprint(device)

# create custom fields and associate custom fields with IP/IPRange/Prefix

@dataclass
class CustomField:
    object_types: list[str]
    type: str
    name: str
    label: str
    description: str
    required: bool
    filter_logic: str

custom_fields = [
    CustomField(
        object_types=["ipam.ipaddress", "ipam.iprange", "ipam.prefix"],
        type="text",
        name="netboxOperatorRestorationHash",
        label="Netbox Restoration Hash",
        description="Used to rediscover previously claimed IP Addresses",
        required=False,
        filter_logic="exact"
    ),
    CustomField(
        object_types=["ipam.ipaddress", "ipam.iprange", "ipam.prefix"],
        type="text",
        name="example_field",
        label="Example Field",
        description="example description",
        required=False,
        filter_logic="exact"
    ),
    CustomField(
        object_types=["ipam.prefix"],
        type="text",
        name="environment",
        label="Environment",
        description="Custom field 1 for ParentPrefixSelector",
        required=False,
        filter_logic="exact"
    ),
    CustomField(
        object_types=["ipam.prefix"],
        type="text",
        name="poolName",
        label="Pool Name",
        description="Custom field 2 for ParentPrefixSelector",
        required=False,
        filter_logic="exact"
    ),
    CustomField(
        object_types=["ipam.prefix"],
        type="boolean",
        name="cfDataTypeBool",
        label="cf Data Type Bool",
        description="Custom field 3 for ParentPrefixSelector",
        required=False,
        filter_logic="exact"
    ),
    CustomField(
        object_types=["ipam.prefix"],
        type="integer",
        name="cfDataTypeInteger",
        label="cf Data Type Integer",
        description="Custom field 4 for ParentPrefixSelector",
        required=False,
        filter_logic="exact"
    ),
]

for custom_field in custom_fields:
    try:
        nb.extras.custom_fields.create(
            object_types=custom_field.object_types,
            type=custom_field.type,
            name=custom_field.name,
            label=custom_field.label,
            description=custom_field.description,
            required=custom_field.required,
            filter_logic=custom_field.filter_logic,
            default=None
        )
    except pynetbox.RequestError as e:
        pprint(e.error)
        sys.exit(1)

# custom_fields = list(nb.extras.custom_fields.all())
# for custom_field in custom_fields:
#     pprint(custom_field)
