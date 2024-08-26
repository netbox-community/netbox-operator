-- create custom fields
INSERT INTO public.extras_customfield (id, type, name, label, description, required, filter_logic, "default", weight, validation_minimum, validation_maximum, validation_regex, created, last_updated, related_object_type_id, group_name, search_weight, is_cloneable, choice_set_id, ui_editable, ui_visible, comments)
VALUES (2, 'text', 'netboxOperatorRestorationHash', 'Netbox Restoration Hash', 'Used to rediscover previously claimed IP Addresses', false, 'exact', NULL, 100, NULL, NULL, '', '2024-06-13 15:17:08.65334+00', '2024-06-13 15:17:08.653359+00', NULL, 'netbox-operator', 100, false, NULL, 'hidden', 'always', '');

INSERT INTO public.extras_customfield (id, type, name, label, description, required, filter_logic, "default", weight, validation_minimum, validation_maximum, validation_regex, created, last_updated, related_object_type_id, group_name, search_weight, is_cloneable, choice_set_id, ui_editable, ui_visible, comments)
VALUES (3, 'text', 'example_field', 'Example Field', 'example description', false, 'exact', NULL, 100, NULL, NULL, '', '2024-06-13 15:17:08.65334+00', '2024-06-13 15:17:08.653359+00', NULL, 'netbox-operator', 100, false, NULL, 'hidden', 'always', '');

-- for IP
INSERT INTO public.extras_customfield_object_types (id, customfield_id, objecttype_id)
VALUES (2, 2, 69);

INSERT INTO public.extras_customfield_object_types (id, customfield_id, objecttype_id)
VALUES (3, 3, 69);

-- for Prefix
INSERT INTO public.extras_customfield_object_types (id, customfield_id, objecttype_id)
VALUES (4, 2, 70);

INSERT INTO public.extras_customfield_object_types (id, customfield_id, objecttype_id)
VALUES (5, 3, 70);

-- misc
INSERT INTO public.tenancy_tenant (created, last_updated, custom_field_data, id, name, slug, description, comments, group_id)
VALUES ('2024-06-14 09:57:11.709344+00', '2024-06-14 09:57:11.709359+00', '{"cust_id": null}', 100, 'MY_TENANT', 'my_tenant', '', '', NULL);
INSERT INTO public.ipam_prefix (created, last_updated, custom_field_data, prefix, status, is_pool, description, role_id, site_id, tenant_id, vlan_id, vrf_id, _children, _depth, mark_utilized, comments)
VALUES ('2024-06-14 10:01:10.094083+00', '2024-06-14 10:01:10.094095+00', '{}', '2.0.0.0/16', 'active', false, '', NULL, NULL, 100, NULL, NULL, 0, 0, false, '');
INSERT INTO public.users_token (id, created, expires, key, write_enabled, description, user_id, allowed_ips, last_used)
VALUES (1, '2024-06-14 12:20:13.317942+00', NULL, '0123456789abcdef0123456789abcdef01234567', true, 'test-token', 1, '{}', NULL);
