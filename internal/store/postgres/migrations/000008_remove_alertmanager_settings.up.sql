-- Remove obsolete alertmanager settings and add SMTP settings for direct email notifications.
-- Alertmanager integration was replaced with direct SMTP email and webhook notifications.

-- Remove obsolete alertmanager setting
DELETE FROM system_settings WHERE key = 'alerts.alertmanager_url';

-- Update TLS setting description (was referencing Alertmanager)
UPDATE system_settings
SET description = 'Skip TLS certificate verification for alert notifications'
WHERE key = 'alerts.tls_insecure_skip_verify';

-- Add SMTP settings for email notifications (if they don't exist)
INSERT INTO system_settings (key, value, value_type, category, description, is_sensitive)
VALUES
    ('alerts.smtp_host', '', 'string', 'alerts', 'SMTP server hostname for sending alert emails', FALSE),
    ('alerts.smtp_port', '587', 'number', 'alerts', 'SMTP server port (typically 587 for STARTTLS, 465 for TLS, 25 for plain)', FALSE),
    ('alerts.smtp_username', '', 'string', 'alerts', 'SMTP authentication username', FALSE),
    ('alerts.smtp_password', '', 'string', 'alerts', 'SMTP authentication password', TRUE),
    ('alerts.smtp_from', '', 'string', 'alerts', 'Email address to send alerts from', FALSE),
    ('alerts.smtp_reply_to', '', 'string', 'alerts', 'Reply-to email address for alerts', FALSE),
    ('alerts.smtp_security', 'starttls', 'string', 'alerts', 'SMTP connection security: none, starttls, or tls', FALSE)
ON CONFLICT (key) DO NOTHING;
