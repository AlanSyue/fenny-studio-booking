-- Default admin user (password: admin123)
-- CHANGE THIS PASSWORD after first login!
INSERT INTO admin_users (username, password_hash)
VALUES ('fenny_studio_super_admin', '$2a$10$m6C5ikZ4YRUfe139XL8E8Ovp.pCxdezxdHfLueHx6HexmiGEr.w6.')
ON CONFLICT (username) DO NOTHING;
