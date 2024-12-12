-- create Custom Fields
INSERT INTO public.extras_customfield (id, type, name, label, description, required, filter_logic, "default", weight, validation_minimum, validation_maximum, validation_regex, created, last_updated, related_object_type_id, group_name, search_weight, is_cloneable, choice_set_id, ui_editable, ui_visible, comments, "unique", related_object_filter)
VALUES (2, 'text', 'netboxOperatorRestorationHash', 'Netbox Restoration Hash', 'Used to rediscover previously claimed IP Addresses', false, 'exact', NULL, 100, NULL, NULL, '', '2024-06-13 15:17:08.65334+00', '2024-06-13 15:17:08.653359+00', NULL, 'netbox-operator', 100, false, NULL, 'hidden', 'always', '', false, NULL);

INSERT INTO public.extras_customfield (id, type, name, label, description, required, filter_logic, "default", weight, validation_minimum, validation_maximum, validation_regex, created, last_updated, related_object_type_id, group_name, search_weight, is_cloneable, choice_set_id, ui_editable, ui_visible, comments, "unique", related_object_filter)
VALUES (3, 'text', 'example_field', 'Example Field', 'example description', false, 'exact', NULL, 100, NULL, NULL, '', '2024-06-13 15:17:08.65334+00', '2024-06-13 15:17:08.653359+00', NULL, 'netbox-operator', 100, false, NULL, 'hidden', 'always', '', false, NULL);

INSERT INTO public.extras_customfield (id, type, name, label, description, required, filter_logic, "default", weight, validation_minimum, validation_maximum, validation_regex, created, last_updated, related_object_type_id, group_name, search_weight, is_cloneable, choice_set_id, ui_editable, ui_visible, comments, "unique", related_object_filter)
VALUES (4, 'text', 'environment', 'Environment', 'Custom field 1 for ParentPrefixSelector', false, 'exact', NULL, 100, NULL, NULL, '', '2024-06-13 15:17:08.65334+00', '2024-06-13 15:17:08.653359+00', NULL, 'netbox-operator', 100, false, NULL, 'hidden', 'always', '', false, NULL);

INSERT INTO public.extras_customfield (id, type, name, label, description, required, filter_logic, "default", weight, validation_minimum, validation_maximum, validation_regex, created, last_updated, related_object_type_id, group_name, search_weight, is_cloneable, choice_set_id, ui_editable, ui_visible, comments, "unique", related_object_filter)
VALUES (5, 'text', 'poolName', 'Pool Name', 'Custom field 2 for ParentPrefixSelector', false, 'exact', NULL, 100, NULL, NULL, '', '2024-06-13 15:17:08.65334+00', '2024-06-13 15:17:08.653359+00', NULL, 'netbox-operator', 100, false, NULL, 'hidden', 'always', '', false, NULL);

INSERT INTO public.extras_customfield (id, type, name, label, description, required, filter_logic, "default", weight, validation_minimum, validation_maximum, validation_regex, created, last_updated, related_object_type_id, group_name, search_weight, is_cloneable, choice_set_id, ui_editable, ui_visible, comments, "unique", related_object_filter)
VALUES (6, 'boolean', 'cfDataTypeBool', 'cf Data Type Bool', 'Custom field 3 for ParentPrefixSelector', false, 'exact', NULL, 100, NULL, NULL, '', '2024-06-13 15:17:08.65334+00', '2024-06-13 15:17:08.653359+00', NULL, 'netbox-operator', 100, false, NULL, 'hidden', 'always', '', false, NULL);

INSERT INTO public.extras_customfield (id, type, name, label, description, required, filter_logic, "default", weight, validation_minimum, validation_maximum, validation_regex, created, last_updated, related_object_type_id, group_name, search_weight, is_cloneable, choice_set_id, ui_editable, ui_visible, comments, "unique", related_object_filter)
VALUES (7, 'integer', 'cfDataTypeInteger', 'cf Data Type Integer', 'Custom field 4 for ParentPrefixSelector', false, 'exact', NULL, 100, NULL, NULL, '', '2024-06-13 15:17:08.65334+00', '2024-06-13 15:17:08.653359+00', NULL, 'netbox-operator', 100, false, NULL, 'hidden', 'always', '', false, NULL);

-- associate custom fields with IP
INSERT INTO public.extras_customfield_object_types (id, customfield_id, objecttype_id)
VALUES (2, 2, 69);

INSERT INTO public.extras_customfield_object_types (id, customfield_id, objecttype_id)
VALUES (3, 3, 69);

-- associate custom fields with Prefix
INSERT INTO public.extras_customfield_object_types (id, customfield_id, objecttype_id)
VALUES (4, 2, 70);

INSERT INTO public.extras_customfield_object_types (id, customfield_id, objecttype_id)
VALUES (5, 3, 70);

INSERT INTO public.extras_customfield_object_types (id, customfield_id, objecttype_id)
VALUES (6, 4, 70);

INSERT INTO public.extras_customfield_object_types (id, customfield_id, objecttype_id)
VALUES (7, 5, 70);

INSERT INTO public.extras_customfield_object_types (id, customfield_id, objecttype_id)
VALUES (8, 6, 70);

INSERT INTO public.extras_customfield_object_types (id, customfield_id, objecttype_id)
VALUES (9, 7, 70);

-- associate custom fields with IP Range
INSERT INTO public.extras_customfield_object_types (id, customfield_id, objecttype_id)
VALUES (10, 2, 78);

INSERT INTO public.extras_customfield_object_types (id, customfield_id, objecttype_id)
VALUES (11, 3, 78);

-- insert User Token
INSERT INTO public.users_token (id, created, expires, key, write_enabled, description, user_id, allowed_ips, last_used)
VALUES (1, '2024-06-14 12:20:13.317942+00', NULL, '0123456789abcdef0123456789abcdef01234567', true, 'test-token', 1, '{}', NULL);

-- insert Tenant
INSERT INTO public.tenancy_tenant (created, last_updated, custom_field_data, id, name, slug, description, comments, group_id)
VALUES ('2024-06-14 09:57:11.709344+00', '2024-06-14 09:57:11.709359+00', '{"cust_id": null}', 100, 'MY_TENANT', 'my_tenant', '', '', NULL);

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
