-- Created by Vertabelo (http://vertabelo.com)
-- Last modification date: 2025-06-17 03:33:52.026

-- foreign keys
ALTER TABLE "vehicle_user_access"
    DROP CONSTRAINT "vehicles_permissions";

ALTER TABLE "vehicles"
    DROP CONSTRAINT "vehicles_users";

ALTER TABLE "maintenance_records"
    DROP CONSTRAINT "maintainance_records_vehicles";

ALTER TABLE "maintenance_records"
    DROP CONSTRAINT "maintenance_records_users";

ALTER TABLE "vehicle_user_access"
    DROP CONSTRAINT "permissions_users";

ALTER TABLE "vehicle_user_access"
    DROP CONSTRAINT "permissions_users_created";

-- tables
DROP TABLE "vehicles";

DROP TABLE "deleted";

DROP TABLE "vehicle_user_access";

DROP TABLE "maintenance_records";

DROP TABLE "users";

DROP FUNCTION update_updated_at;;

-- End of file.

