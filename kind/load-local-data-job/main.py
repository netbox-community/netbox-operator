import os
import pynetbox
from pprint import pprint
from dataclasses import dataclass
from typing import Optional

print("Starting to load data onto NetBox through API")

NETBOX_API = os.getenv("NETBOX_API", "http://netbox")

try:
    nb = pynetbox.api(
        NETBOX_API,
        token='0123456789abcdef0123456789abcdef01234567'
    )
except pynetbox.RequestError as e:
    pprint(e.error)
    raise SystemExit(f"Failed to connect to NetBox at {NETBOX_API}")

print(f"Connected to NetBoxAPI at {NETBOX_API}")


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
    Tenant(
        name="MY_TENANT_2",
        slug="my_tenant_2",
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

print("Tenants loaded")

# insert Sites
@dataclass
class Site:
    name: str
    slug: str
    status: str
    tenant: dict

sites = [
    Site(
        name="MY_SITE",
        slug="my_site",
        status="active",
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
    ),
    Site(
        name="MY_SITE_2",
        slug="my_site_2",
        status="active",
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
    ),
]

for site in sites:
    try:
        nb.dcim.sites.create(
            name=site.name,
            slug=site.slug,
            tenant=site.tenant,
        )
    except pynetbox.RequestError as e:
        pprint(e.error)

print("Sites loaded")

# create custom fields and associate custom fields with IP/IPRange/Prefix
@dataclass
class CustomField:
    content_types: list[str] # for v3
    object_types: list[str] # for v4
    type: str
    name: str
    label: str
    description: str
    required: bool
    filter_logic: str
    validation_minimum: Optional[int] = None
    validation_maximum: Optional[int] = None

custom_fields = [
    CustomField(
        content_types=["ipam.ipaddress", "ipam.iprange", "ipam.prefix", "ipam.asn"],
        object_types=["ipam.ipaddress", "ipam.iprange", "ipam.prefix", "ipam.asn"],
        type="text",
        name="netboxOperatorRestorationHash",
        label="Netbox Restoration Hash",
        description="Used to rediscover previously claimed IP Addresses",
        required=False,
        filter_logic="exact"
    ),
    CustomField(
        content_types=["ipam.ipaddress", "ipam.iprange", "ipam.prefix", "ipam.asn"],
        object_types=["ipam.ipaddress", "ipam.iprange", "ipam.prefix", "ipam.asn"],
        type="text",
        name="example_field",
        label="Example Field",
        description="example description",
        required=False,
        filter_logic="exact"
    ),
    CustomField(
        content_types=["ipam.prefix"],
        object_types=["ipam.prefix"],
        type="text",
        name="environment",
        label="Environment",
        description="Custom field 1 for ParentPrefixSelector",
        required=False,
        filter_logic="exact"
    ),
    CustomField(
        content_types=["ipam.prefix"],
        object_types=["ipam.prefix"],
        type="text",
        name="poolName",
        label="Pool Name",
        description="Custom field 2 for ParentPrefixSelector",
        required=False,
        filter_logic="exact"
    ),
    CustomField(
        content_types=["ipam.prefix"],
        object_types=["ipam.prefix"],
        type="boolean",
        name="cfDataTypeBool",
        label="cf Data Type Bool",
        description="Custom field 3 for ParentPrefixSelector",
        required=False,
        filter_logic="exact"
    ),
    CustomField(
        content_types=["ipam.prefix"],
        object_types=["ipam.prefix"],
        type="integer",
        name="cfDataTypeInteger",
        label="cf Data Type Integer",
        description="Custom field 4 for ParentPrefixSelector",
        required=False,
        filter_logic="exact"
    ),
    CustomField(
        content_types=["ipam.prefix"],
        object_types=["ipam.prefix"],
        type="integer",
        name="cfDataTypeIntegerValidationRange",
        label="cf Data Type Integer Validation Range",
        description="Custom field with integer validation bounds",
        required=False,
        filter_logic="exact",
        validation_minimum=0,
        validation_maximum=10,
    ),
]

for custom_field in custom_fields:
    try:
        nb.extras.custom_fields.create(
            content_types=custom_field.content_types,
            object_types=custom_field.object_types,
            type=custom_field.type,
            name=custom_field.name,
            label=custom_field.label,
            description=custom_field.description,
            required=custom_field.required,
            filter_logic=custom_field.filter_logic,
            validation_minimum=custom_field.validation_minimum,
            validation_maximum=custom_field.validation_maximum,
            default=None
        )
    except pynetbox.RequestError as e:
        pprint(e.error)

print("Custom fields loaded")

# for debugging
# custom_fields = list(nb.extras.custom_fields.all())
# for custom_field in custom_fields:
#     pprint(custom_field)

# insert Prefixes
@dataclass
class Prefix:
    prefix: str
    site: dict
    scope_id: int
    scope_type: str
    tenant: dict
    status: str
    custom_fields: dict
    description: str

scopeId = nb.dcim.sites.get(name="MY_SITE").id

# Note: 4.0.0.0/8 (4.x.y.z) prefixes are used by e2e tests that dynamically create the test data using Prefix custom resources.
prefixes = [
    Prefix(
        prefix="2.0.0.0/16",
        description="",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "Dunder-Mifflin, Inc.",
            "slug": "dunder-mifflin",
        },
        status="active",
        custom_fields={},
    ),

    ###                     START                   ###
    ###                Used by e2e tests            ###
    ### Modifying entries might cause tests to fail ###
    # Resources used by Prefix and PrefixClaim tests
    Prefix(
        prefix="2.0.1.0/24",
        description="chainsaw test prefixclaim-ipv4-prefixexhausted",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={},
    ),
    Prefix(
        prefix="2.0.2.0/24",
        description="chainsaw test prefixclaim-ipv4-apply",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={},
    ),
    Prefix(
        prefix="2.0.3.0/24",
        description="chainsaw test prefixclaim-ipv4-parentprefixselector-restore",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={},
    ),
    Prefix( # TODO(henrybear327): debug why prefixclaim-ipv4-parentprefixselector-apply-succeed isn't using this prefix
        prefix="3.0.0.0/24",
        description="",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "Production",
            "poolName": "Pool 1",
            "cfDataTypeBool": True,
            "cfDataTypeInteger": 1,
        },
    ),
    Prefix(
        prefix="3.0.1.0/24",
        description="chainsaw test prefixclaim-ipv4-parentprefixselector",
        site={
            "name": "MY_SITE",
            "slug": "my_site",
            "tenant": {
                "name": "MY_TENANT",
                "slug": "my_tenant",
            },
        },
        scope_id=scopeId,
        scope_type="dcim.site",
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "Production",
            "poolName": "Pool 1",
            "cfDataTypeBool": True,
            "cfDataTypeInteger": 1,
        },
    ),
    Prefix(
        prefix="3.0.2.0/24",
        description="",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "Development",
            "poolName": "Pool 1",
            "cfDataTypeBool": False,
            "cfDataTypeInteger": 2,
        },
    ),
    Prefix( # TODO(henrybear327): debug why prefixclaim-ipv4-parentprefixselector-restoration-succeed isn't using this prefix
        prefix="3.0.3.0/24",
        description="",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "Production",
            "poolName": "Pool 2",
            "cfDataTypeBool": True,
            "cfDataTypeInteger": 3,
        },
    ),
    Prefix(
        prefix="3.0.4.0/24",
        description="chainsaw test prefixclaim-ipv4-restore",
        site={
            "name": "MY_SITE",
            "slug": "my_site",
            "tenant": {
                "name": "MY_TENANT",
                "slug": "my_tenant",
            },
        },
        scope_id=scopeId,
        scope_type="dcim.site",
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "Production",
            "poolName": "Pool 2",
            "cfDataTypeBool": True,
            "cfDataTypeInteger": 3,
        },
    ),
    Prefix(
        prefix="3.0.5.0/24",
        description="chainsaw test prefixclaim-ipv4-update-ownerreference",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "Development",
            "poolName": "Pool 2",
            "cfDataTypeBool": False,
            "cfDataTypeInteger": 4,
        },
    ),
    Prefix(
        prefix="3.0.6.0/24",
        description="",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "production",
            "poolName": "pool 3",
            "cfDataTypeBool": True,
            "cfDataTypeInteger": 5,
        },
    ),
    Prefix(
        prefix="3.0.7.0/24",
        description="",
        site={
            "name": "MY_SITE",
            "slug": "my_site",
            "tenant": {
                "name": "MY_TENANT",
                "slug": "my_tenant",
            },
        },
        scope_id=scopeId,
        scope_type="dcim.site",
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "production",
            "poolName": "pool 3",
            "cfDataTypeBool": True,
            "cfDataTypeInteger": 5,
        },
    ),
    Prefix(
        prefix="3.0.8.0/24",
        description="",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "development",
            "poolName": "pool 3",
            "cfDataTypeBool": False,
            "cfDataTypeInteger": 6,
        },
    ),
    Prefix(
        prefix="2::/64",
        description="chainsaw test prefixclaim-ipv6-apply-update",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "production",
            "poolName": "pool 4",
            "cfDataTypeBool": True,
            "cfDataTypeInteger": 7,
        },
    ),
    Prefix( # TODO(henrybear327): debug why this entry is missing from NetBox after chainsaw test execution
        prefix="2:0:0:1::/64",
        description="",
        site={
            "name": "MY_SITE",
            "slug": "my_site",
            "tenant": {
                "name": "MY_TENANT",
                "slug": "my_tenant",
            },
        },
        scope_id=scopeId,
        scope_type="dcim.site",
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "production",
            "poolName": "pool 4",
            "cfDataTypeBool": True,
            "cfDataTypeInteger": 7,
        },
    ),
    Prefix(
        prefix="2:0:0:2::/64",
        description="",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "development",
            "poolName": "pool 4",
            "cfDataTypeBool": False,
            "cfDataTypeInteger": 8,
        },
    ),
    # Resources used by IpAddress and IpAddressClaim tests
    Prefix(
        prefix="3.1.0.0/24",
        description="chainsaw test ipaddressclaim-ipv4-apply-update",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "development",
            "poolName": "pool 3",
            "cfDataTypeBool": False,
            "cfDataTypeInteger": 7,
        },
    ),
    Prefix(
        prefix="3.1.1.0/30",
        description="chainsaw test ipaddressclaim-ipv4-prefixexhausted",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "development",
            "poolName": "pool 3",
            "cfDataTypeBool": False,
            "cfDataTypeInteger": 8,
        },
    ),
    Prefix(
        prefix="3.1.2.0/24",
        description="chainsaw test ipaddressclaim-ipv4-restore",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "development",
            "poolName": "pool 3",
            "cfDataTypeBool": False,
            "cfDataTypeInteger": 7,
        },
    ),
    Prefix(
        prefix="3.1.3.0/24",
        description="chainsaw test ipaddressclaim-ipv4-update-ownerreference",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "development",
            "poolName": "pool 3",
            "cfDataTypeBool": False,
            "cfDataTypeInteger": 7,
        },
    ),
    Prefix(
        prefix="3.1.4.0/24",
        description="",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "development",
            "poolName": "pool 3",
            "cfDataTypeBool": False,
            "cfDataTypeInteger": 7,
        },
    ),
    Prefix(
        prefix="3.1.5.0/24",
        description="",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "development",
            "poolName": "pool 3",
            "cfDataTypeBool": False,
            "cfDataTypeInteger": 7,
        },
    ),
    Prefix(
        prefix="3.1.6.0/24",
        description="",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "development",
            "poolName": "pool 3",
            "cfDataTypeBool": False,
            "cfDataTypeInteger": 7,
        },
    ),
    Prefix(
        prefix="3.1.7.0/24",
        description="",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "development",
            "poolName": "pool 3",
            "cfDataTypeBool": False,
            "cfDataTypeInteger": 7,
        },
    ),
    Prefix(
        prefix="3.1.8.0/24",
        description="",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "development",
            "poolName": "pool 3",
            "cfDataTypeBool": False,
            "cfDataTypeInteger": 7,
        },
    ),
    Prefix(
        prefix="3.1.9.0/24",
        description="",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "development",
            "poolName": "pool 3",
            "cfDataTypeBool": False,
            "cfDataTypeInteger": 7,
        },
    ),
    Prefix(
        prefix="3:1:0::/64",
        description="chainsaw test ipaddressclaim-ipv6-apply-update",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "development",
            "poolName": "pool 3",
            "cfDataTypeBool": False,
            "cfDataTypeInteger": 7,
        },
    ),
    Prefix(
        prefix="3:1:1::/127",
        description="chainsaw test ipaddressclaim-ipv6-prefixexhausted",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "development",
            "poolName": "pool 3",
            "cfDataTypeBool": False,
            "cfDataTypeInteger": 7,
        },
    ),
    Prefix(
        prefix="3:1:2::/64",
        description="chainsaw test ipaddressclaim-ipv6-restore",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "development",
            "poolName": "pool 3",
            "cfDataTypeBool": False,
            "cfDataTypeInteger": 7,
        },
    ),
    Prefix(
        prefix="3:1:3::/64",
        description="",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "development",
            "poolName": "pool 3",
            "cfDataTypeBool": False,
            "cfDataTypeInteger": 7,
        },
    ),
    # Resources used by IpRange and IpRangeClaim tests
    Prefix(
        prefix="3.2.0.0/24",
        description="chainsaw test iprangeclaim-ipv4-apply-update",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "development",
            "poolName": "pool 3",
            "cfDataTypeBool": False,
            "cfDataTypeInteger": 7,
        },
    ),
    Prefix(
        prefix="3.2.1.0/26",
        description="chainsaw test iprangeclaim-ipv4-prefixexhausted",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "development",
            "poolName": "pool 3",
            "cfDataTypeBool": False,
            "cfDataTypeInteger": 8,
        },
    ),
    Prefix(
        prefix="3.2.2.0/24",
        description="chainsaw test iprangeclaim-ipv4-restore",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "development",
            "poolName": "pool 3",
            "cfDataTypeBool": False,
            "cfDataTypeInteger": 7,
        },
    ),
    Prefix(
        prefix="3.2.3.0/24",
        description="chainsaw test iprangeclaim-ipv4-invalid-*",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "development",
            "poolName": "pool 3",
            "cfDataTypeBool": False,
            "cfDataTypeInteger": 7,
        },
    ),
    Prefix(
        prefix="3.2.4.0/24",
        description="chainsaw test iprangeclaim-ipv4-update-ownerreference",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "development",
            "poolName": "pool 3",
            "cfDataTypeBool": False,
            "cfDataTypeInteger": 7,
        },
    ),
    Prefix(
        prefix="3.2.5.0/24",
        description="",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "development",
            "poolName": "pool 3",
            "cfDataTypeBool": False,
            "cfDataTypeInteger": 7,
        },
    ),
    Prefix(
        prefix="3.2.6.0/24",
        description="",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "development",
            "poolName": "pool 3",
            "cfDataTypeBool": False,
            "cfDataTypeInteger": 7,
        },
    ),
    Prefix(
        prefix="3.2.7.0/24",
        description="",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "development",
            "poolName": "pool 3",
            "cfDataTypeBool": False,
            "cfDataTypeInteger": 7,
        },
    ),
    Prefix(
        prefix="3.2.8.0/24",
        description="",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "development",
            "poolName": "pool 3",
            "cfDataTypeBool": False,
            "cfDataTypeInteger": 7,
        },
    ),
    Prefix(
        prefix="3.2.9.0/24",
        description="",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "development",
            "poolName": "pool 3",
            "cfDataTypeBool": False,
            "cfDataTypeInteger": 7,
        },
    ),
    Prefix(
        prefix="3:2:0::/64",
        description="chainsaw test iprangeclaim-ipv6-apply-update",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "development",
            "poolName": "pool 3",
            "cfDataTypeBool": False,
            "cfDataTypeInteger": 7,
        },
    ),
    Prefix(
        prefix="3:2:1::/122",
        description="chainsaw test iprangeclaim-ipv6-prefixexhausted",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "development",
            "poolName": "pool 3",
            "cfDataTypeBool": False,
            "cfDataTypeInteger": 7,
        },
    ),
    Prefix(
        prefix="3:2:2::/64",
        description="chainsaw test iprangeclaim-ipv6-restore",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "development",
            "poolName": "pool 3",
            "cfDataTypeBool": False,
            "cfDataTypeInteger": 7,
        },
    ),
    Prefix(
        prefix="3:2:3::/64",
        description="",
        site=None,
        scope_id=None,
        scope_type=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={
            "environment": "development",
            "poolName": "pool 3",
            "cfDataTypeBool": False,
            "cfDataTypeInteger": 7,
        },
    ),
    ###                      END                    ###
    ###                Used by e2e tests            ###
    ### Modifying entries might cause tests to fail ###
]

for prefix in prefixes:
    try:
        nb.ipam.prefixes.create(
            prefix=prefix.prefix,
            site=prefix.site,
            scope_type=prefix.scope_type,
            scope_id=prefix.scope_id,
            description=prefix.description,
            tenant=prefix.tenant,
            status=prefix.status,
            custom_fields=prefix.custom_fields,
        )
    except pynetbox.RequestError as e:
        pprint(e.error)

print("Prefixes loaded")

# insert RIR (required for ASN Ranges)
try:
    nb.ipam.rirs.create(
        name="E2E Test RIR",
        slug="e2e-test-rir",
        is_private=True,
    )
except pynetbox.RequestError as e:
    pprint(e.error)

print("RIRs loaded")

# insert ASN Ranges
rir = nb.ipam.rirs.get(name="E2E Test RIR")

@dataclass
class AsnRange:
    name: str
    slug: str
    start: int
    end: int
    rir: int
    tenant: Optional[dict]
    description: str

asn_ranges = [
    ###                     START                   ###
    ###                Used by e2e tests            ###
    ### Modifying entries might cause tests to fail ###
    AsnRange(
        name="E2E Test ASN Range",
        slug="e2e-test-asn-range",
        start=64512,
        end=64612,
        rir=rir.id,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        description="chainsaw test asnclaim-apply-update",
    ),
    AsnRange(
        name="E2E Test ASN Range Restore",
        slug="e2e-test-asn-range-restore",
        start=64700,
        end=64800,
        rir=rir.id,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        description="chainsaw test asnclaim-restore",
    ),
    AsnRange(
        # Keep clear of 65001-65005, which the NetBox demo data already occupies.
        name="E2E Test ASN Range Exhausted",
        slug="e2e-test-asn-range-exhausted",
        start=65300,
        end=65301,
        rir=rir.id,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        description="chainsaw test asnclaim-asnrangeexhausted",
    ),
    AsnRange(
        name="E2E Test ASN Range OwnerRef",
        slug="e2e-test-asn-range-ownerref",
        start=65100,
        end=65200,
        rir=rir.id,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        description="chainsaw test asnclaim-update-ownerreference",
    ),
    AsnRange(
        name="E2E Test ASN Range 32bit",
        slug="e2e-test-asn-range-32bit",
        start=4200000000,
        end=4200000100,
        rir=rir.id,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        description="chainsaw test asnclaim-32bit",
    ),
    ###                      END                    ###
    ###                Used by e2e tests            ###
    ### Modifying entries might cause tests to fail ###
]

for asn_range in asn_ranges:
    try:
        nb.ipam.asn_ranges.create(
            name=asn_range.name,
            slug=asn_range.slug,
            start=asn_range.start,
            end=asn_range.end,
            rir=asn_range.rir,
            tenant=asn_range.tenant,
            description=asn_range.description,
        )
    except pynetbox.RequestError as e:
        pprint(e.error)

print("ASN Ranges loaded")

###                     START                   ###
###                Used by e2e tests            ###
### Modifying entries might cause tests to fail ###
# The operator lists ASNs with a page size of 250 (asnListPageSize in
# pkg/netbox/api/asn.go) when it looks up an ASN by restoration hash. These filler
# ASNs push the total number of ASNs beyond a single page so that the e2e tests
# exercise the pagination logic. They use low ASN values on purpose: NetBox orders
# ASNs by their value, so the ASNs used by the tests (64512 and above) end up on a
# later page.
ASN_FILLER_START = 1000
ASN_FILLER_COUNT = 400

try:
    nb.ipam.asns.create(
        [
            {
                "asn": asn,
                "rir": rir.id,
                "description": "filler ASN to force pagination in e2e tests",
            }
            for asn in range(ASN_FILLER_START, ASN_FILLER_START + ASN_FILLER_COUNT)
        ]
    )
except pynetbox.RequestError as e:
    pprint(e.error)
###                      END                    ###
###                Used by e2e tests            ###
### Modifying entries might cause tests to fail ###

print("ASNs loaded")
