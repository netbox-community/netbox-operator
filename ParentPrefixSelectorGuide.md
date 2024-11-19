# A guide of `ParentPrefixSelector` in `PrefixClaim`

There are 2 ways to make a Prefix claim:
- provide a `parentPrefix`
- provide a `parentPrefixSelector`

In this documentation, we will focus on the `parentPrefixSelector` only.

# CRD format

The following is a sample of utilizing the `parentPrefixSelector`:

```bash
apiVersion: netbox.dev/v1
kind: PrefixClaim
metadata:
  labels:
    app.kubernetes.io/name: netbox-operator
    app.kubernetes.io/managed-by: kustomize
  name: prefixclaim-customfields-sample
spec:
  tenant: "MY_TENANT"
  site: "DM-Akron"
  description: "some description"
  comments: "your comments"
  preserveInNetbox: true
  prefixLength: "/31"
  parentPrefixSelector: 
    tenant: "MY_TENANT"
    site: "DM-Buffalo"
    environment: "Production"
    poolName: "Pool 1"
```

The usage will be explained in the following sections.

## Notes on `Spec.tenant` and `Spec.site`

Please provide the *name*, not the *slug* value

## `parentPrefixSelector`

The `parentPrefixSelector` is a key-value map, where all the entries are of data type `<string-string>`.

The map contains a set of query conditions for selecting a set of prefixes that can be used as the parent prefix.

The query conditions will be chained by the AND operator, and exact match of the keys and values will be performed.

The fields that can be used as query conditions in the `parentPrefixSelector` are:
- `tenant` and `site` (in lowercase characters)
    - these 2 fields are built-in fields from NetBox, so you do *not* need to create custom fields for them
    - please provide the *name*, not the *slug* value
    - if the entry for tenant or site is missing, it will *not* inherit from the tenant and site from the Spec
- custom fields
    - the data types tested and supported so far are `string`, `integer`, and `boolean`
    - for `boolean` type, please use `true` and `false` as the value
