import os
import pynetbox
from pprint import pprint
from dataclasses import dataclass

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

custom_fields = [
    CustomField(
        content_types=["ipam.ipaddress", "ipam.iprange", "ipam.prefix"],
        object_types=["ipam.ipaddress", "ipam.iprange", "ipam.prefix"],
        type="text",
        name="netboxOperatorRestorationHash",
        label="Netbox Restoration Hash",
        description="Used to rediscover previously claimed IP Addresses",
        required=False,
        filter_logic="exact"
    ),
    CustomField(
        content_types=["ipam.ipaddress", "ipam.iprange", "ipam.prefix"],
        object_types=["ipam.ipaddress", "ipam.iprange", "ipam.prefix"],
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
