import pynetbox
from pprint import pprint
from dataclasses import dataclass
import sys

print("Starting to load data onto NetBox through API")
try:
    nb = pynetbox.api(
        'http://netbox',
        token='0123456789abcdef0123456789abcdef01234567'
    )
except pynetbox.RequestError as e:
    pprint(e.error)
    sys.exit(1)
print("Connected to NetBoxAPI")

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
        sys.exit(1)
print("Sites loaded")

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
    tenant: dict
    status: str
    custom_fields: dict 

prefixes = [
    Prefix(
        prefix="2.0.0.0/24",    
        site=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={},
    ),
    Prefix(
        prefix="2.0.1.0/24",    
        site=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={},
    ),
    Prefix(
        prefix="2.0.2.0/24",    
        site=None,
        tenant={
            "name": "MY_TENANT",
            "slug": "my_tenant",
        },
        status="active",
        custom_fields={},
    ),

    Prefix(
        prefix="3.0.0.0/24",    
        site=None,
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
        site={
            "name": "MY_SITE",
            "slug": "my_site",
            "tenant": {
                "name": "MY_TENANT",
                "slug": "my_tenant",
            },
        },
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
        site=None,
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
    Prefix(
        prefix="3.0.3.0/24",    
        site=None,
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
        site={
            "name": "MY_SITE",
            "slug": "my_site",
            "tenant": {
                "name": "MY_TENANT",
                "slug": "my_tenant",
            },
        },
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
        site=None,
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
        site=None,
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
        site={
            "name": "MY_SITE",
            "slug": "my_site",
            "tenant": {
                "name": "MY_TENANT",
                "slug": "my_tenant",
            },
        },
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
        site=None,
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
        site=None,
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
        prefix="2:0:0:1::/64",    
        site={
            "name": "MY_SITE",
            "slug": "my_site",
            "tenant": {
                "name": "MY_TENANT",
                "slug": "my_tenant",
            },
        },
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
        site=None,
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
]

for prefix in prefixes:
    try:
        nb.ipam.prefixes.create(
            prefix=prefix.prefix,
            site=prefix.site,
            tenant=prefix.tenant,
            status=prefix.status,
            custom_fields=prefix.custom_fields,
        )
    except pynetbox.RequestError as e:
        pprint(e.error)
        sys.exit(1)
print("Prefixes loaded")
