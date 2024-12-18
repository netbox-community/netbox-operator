-- insert User Token
INSERT INTO public.users_token (id, created, expires, key, write_enabled, description, user_id, allowed_ips, last_used)
VALUES (1, '2024-06-14 12:20:13.317942+00', NULL, '0123456789abcdef0123456789abcdef01234567', true, 'test-token', 1, '{}', NULL);
