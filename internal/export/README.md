# Auto Myself data export

This archive contains a copy of the data in your Auto Myself account, generated when you requested
the export.

## What is in this archive

- `user.json` — your profile and the email addresses linked through Google or Apple sign-in.
- `vehicles/<vehicle-id>/vehicle.json` — a vehicle you own or that has been shared with you.
- `vehicles/<vehicle-id>/maintenance_records.json` — that vehicle's maintenance history.
- `vehicles/<vehicle-id>/files/<file-id>.json` — details for a file attached to a maintenance record.
- `reports/<report-id>.json` — a report you submitted.

All documents are JSON. Deleted items are not included, and the export is a snapshot: changes made
after it was generated are not reflected.

## Downloading attached files

File bytes are not stored inside this archive. Each file's JSON document contains a `signed_url`
link that downloads the file directly.

**Each signed URL is valid for 24 hours from the time this export was generated.** Download what
you need before it expires. After that the link stops working; request a new export to get fresh
links.

Treat these links as private: anyone with a valid link can download the file until it expires.

## What is not included

For your privacy and security, the archive does not contain account credentials, other users'
usernames or personal details, share invitations, or internal moderation notes.
