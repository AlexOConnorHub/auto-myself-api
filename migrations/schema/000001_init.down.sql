ALTER TABLE "vehicle_user_access_pending"
    DROP CONSTRAINT "vehicles_vehicle_user_access_pending";

ALTER TABLE "vehicle_user_access"
    DROP CONSTRAINT "vehicles_vehicle_user_access";

ALTER TABLE "vehicles"
    DROP CONSTRAINT "vehicles_users";

ALTER TABLE "maintenance_records"
    DROP CONSTRAINT "maintenance_records_vehicles";

ALTER TABLE "maintenance_records"
    DROP CONSTRAINT "maintenance_records_users";

ALTER TABLE "vehicle_user_access_pending"
    DROP CONSTRAINT "vehicle_user_access_pending_users";

ALTER TABLE "vehicle_user_access_pending"
    DROP CONSTRAINT "vehicle_user_access_pending_users_created";

ALTER TABLE "vehicle_user_access"
    DROP CONSTRAINT "vehicle_user_access_users";

ALTER TABLE "vehicle_user_access"
    DROP CONSTRAINT "vehicle_user_access_users_created";

ALTER TABLE "identities"
    DROP CONSTRAINT "identities_pk";

DROP TABLE "vehicles";

DROP TABLE "deleted";

DROP TABLE "vehicle_user_access";

DROP TABLE "vehicle_user_access_pending";

DROP TABLE "maintenance_records";

DROP TABLE "users";

DROP TABLE "identities";

DROP FUNCTION update_updated_at;;
