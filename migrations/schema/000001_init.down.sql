-- Created by Vertabelo (http://vertabelo.com)
-- Last modification date: 2025-06-17 03:33:52.026

-- foreign keys
ALTER TABLE "car_user_access"
    DROP CONSTRAINT "cars_permissions";

ALTER TABLE "cars"
    DROP CONSTRAINT "cars_users";

ALTER TABLE "maintenance_records"
    DROP CONSTRAINT "maintainance_records_cars";

ALTER TABLE "maintenance_records"
    DROP CONSTRAINT "maintenance_records_users";

ALTER TABLE "car_user_access"
    DROP CONSTRAINT "permissions_users";

ALTER TABLE "car_user_access"
    DROP CONSTRAINT "permissions_users_created";

-- tables
DROP TABLE "cars";

DROP TABLE "deleted";

DROP TABLE "car_user_access";

DROP TABLE "maintenance_records";

DROP TABLE "users";

DROP FUNCTION update_updated_at;;

-- End of file.

