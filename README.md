# Auto Myself API

This is a web API which integrates with the Auto Myself application to allow for backup and sharing of vehicles and their maintenance.

## Setup

### Recomeneded user/database Setup

1. Reconnect to the server, specifying the database and then create the users
    - ```
        CREATE ROLE auto_myself_admin WITH LOGIN PASSWORD 'replace-password' CREATEDB CREATEROLE;
        CREATE ROLE auto_myself_api WITH LOGIN PASSWORD 'replace-password';
        CREATE DATABASE auto_myself OWNER auto_myself_admin;

        \c auto_myself
        REVOKE ALL ON SCHEMA public FROM PUBLIC;

        GRANT CONNECT ON DATABASE auto_myself TO auto_myself_api;
        GRANT USAGE ON SCHEMA public TO auto_myself_api;
        GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA public TO auto_myself_api;
        GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO auto_myself_api;

        ALTER DEFAULT PRIVILEGES FOR ROLE auto_myself_admin IN SCHEMA public GRANT SELECT, INSERT, UPDATE ON TABLES TO auto_myself_api;
        ALTER DEFAULT PRIVILEGES FOR ROLE auto_myself_admin IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO auto_myself_api;
        ```
