# Auto Myself API

This is a web API which integrates with the Auto Myself application to allow for backup and sharing of vehicles and their maintenance.

## Setup

### Recomeneded user/database Setup

1. Create the database
   - ```
        create database "auto-myself-api";
        ```
1. Reconnect to the server, specifying the database and then create the users
    - ```
        CREATE ROLE "app-user" LOGIN PASSWORD 'abc';
        CREATE ROLE "schema-admin" LOGIN PASSWORD 'abc';

        REVOKE ALL ON SCHEMA public FROM PUBLIC;
        REVOKE ALL ON DATABASE "auto-myself-api" FROM PUBLIC;

        GRANT CREATE, USAGE ON SCHEMA public TO "schema-admin";

        GRANT USAGE ON SCHEMA public TO "app-user";
        GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA public TO "app-user";

        ALTER DEFAULT PRIVILEGES IN SCHEMA public
        GRANT SELECT, INSERT, UPDATE ON TABLES TO "app-user";

        GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO "schema-admin";
        ALTER DEFAULT PRIVILEGES IN SCHEMA public
        GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO "schema-admin";

        GRANT CONNECT ON DATABASE "auto-myself-api" TO "schema-admin";
        GRANT CONNECT ON DATABASE "auto-myself-api" TO "app-user";
        ```
