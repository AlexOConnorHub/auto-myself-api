CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER
SET search_path = public
AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TABLE "vehicle_user_access_pending" (
    "id" uuid  NOT NULL,
    "user_id" uuid  NOT NULL,
    "vehicle_id" uuid  NOT NULL,
    "write_access" boolean  NOT NULL DEFAULT false,
    "created_by" uuid  NOT NULL,
    "created_at" timestamptz  NOT NULL DEFAULT now(),
    "updated_at" timestamptz  NOT NULL DEFAULT now(),
    "deleted_at" timestamptz  NULL,
    CONSTRAINT "vehicle_user_access_pending_pk" PRIMARY KEY ("id")
);

CREATE INDEX "vehicle_user_access_pending_idx_1" on "vehicle_user_access_pending" ("vehicle_id" ASC);

CREATE INDEX "vehicle_user_access_pending_idx_2" on "vehicle_user_access_pending" ("user_id" ASC);

CREATE INDEX "vehicle_user_access_pending_idx_3" on "vehicle_user_access_pending" ("created_by" ASC);

CREATE TRIGGER update_vehicle_user_access_pending_updated_at
   BEFORE UPDATE ON "vehicle_user_access_pending"
   FOR EACH ROW EXECUTE PROCEDURE update_updated_at();

CREATE TABLE "vehicle_user_access" (
    "id" uuid  NOT NULL,
    "user_id" uuid  NOT NULL,
    "vehicle_id" uuid  NOT NULL,
    "write_access" boolean  NOT NULL DEFAULT false,
    "created_by" uuid  NOT NULL,
    "created_at" timestamptz  NOT NULL DEFAULT now(),
    "updated_at" timestamptz  NOT NULL DEFAULT now(),
    "deleted_at" timestamptz  NULL,
    CONSTRAINT "vehicle_user_access_pk" PRIMARY KEY ("id")
);

CREATE INDEX "vehicle_user_access_idx_1" on "vehicle_user_access" ("vehicle_id" ASC);

CREATE INDEX "vehicle_user_access_idx_2" on "vehicle_user_access" ("user_id" ASC);

CREATE INDEX "vehicle_user_access_idx_3" on "vehicle_user_access" ("created_by" ASC);

CREATE TRIGGER update_vehicle_user_access_updated_at
   BEFORE UPDATE ON "vehicle_user_access"
   FOR EACH ROW EXECUTE PROCEDURE update_updated_at();

CREATE TABLE "vehicles" (
    "id" uuid  NOT NULL,
    "make" text  NULL,
    "make_id" integer  NULL,
    "model" text  NULL,
    "model_id" integer  NULL,
    "year" integer  NULL,
    "vin" text  NULL,
    "lpn" text  NULL,
    "nickname" text  NULL,
    "created_by" uuid  NOT NULL,
    "created_at" timestamptz  NOT NULL DEFAULT now(),
    "updated_at" timestamptz  NOT NULL DEFAULT now(),
    "deleted_at" timestamptz  NULL,
    CONSTRAINT "vehicle_pk" PRIMARY KEY ("id")
);

CREATE INDEX "vehicles_idx_1" on "vehicles" ("created_by" ASC);

CREATE TRIGGER update_vehicles_updated_at
    BEFORE UPDATE ON "vehicles"
   FOR EACH ROW EXECUTE PROCEDURE update_updated_at();

CREATE TABLE "deleted" (
    "id" uuid  NOT NULL,
    "source_table" text  NOT NULL,
    "source_id" uuid  NOT NULL,
    "deleted_at" timestamptz  NOT NULL,
    CONSTRAINT "deleted_pk" PRIMARY KEY ("id")
);

CREATE INDEX "deleted_idx_1" on "deleted" ("source_table" ASC,"source_id" ASC);

CREATE TABLE "maintenance_records" (
    "id" uuid  NOT NULL,
    "vehicle_id" uuid  NOT NULL,
    "odometer" integer  NULL,
    "timestamp" date  NULL,
    "notes" text  NULL,
    "type" text  NULL,
    "interval" integer  NULL,
    "interval_type" text  NULL,
    "cost" text NULL,
    "created_by" uuid  NOT NULL,
    "created_at" timestamptz  NOT NULL DEFAULT now(),
    "updated_at" timestamptz  NOT NULL DEFAULT now(),
    "deleted_at" timestamptz  NULL,
    CONSTRAINT "maintenance_records_pk" PRIMARY KEY ("id")
);

CREATE INDEX "maintenance_records_idx_1" on "maintenance_records" ("vehicle_id" ASC);

CREATE INDEX "maintenance_records_idx_2" on "maintenance_records" ("created_by" ASC);

CREATE TRIGGER update_maintenance_records_updated_at
   BEFORE UPDATE ON "maintenance_records"
   FOR EACH ROW EXECUTE PROCEDURE update_updated_at();

CREATE TABLE "users" (
    "id" uuid  NOT NULL,
    "username" text  NOT NULL,
    "created_at" timestamptz  NOT NULL DEFAULT now(),
    "updated_at" timestamptz  NOT NULL DEFAULT now(),
    "deleted_at" timestamptz  NULL,
    CONSTRAINT "users_pk" PRIMARY KEY ("id")
);

CREATE TRIGGER update_users_updated_at
   BEFORE UPDATE ON "users"
   FOR EACH ROW EXECUTE PROCEDURE update_updated_at();

CREATE TABLE "identities" (
    "id" uuid  NOT NULL,
    "provider" text  NOT NULL,
    "subject" text  NOT NULL,
    "user_id" uuid  NOT NULL,
    "email" text  NULL,
    "created_at" timestamptz  NOT NULL DEFAULT now(),
    "updated_at" timestamptz  NOT NULL DEFAULT now(),
    "deleted_at" timestamptz  NULL,
    CONSTRAINT "identities_pk" PRIMARY KEY ("id")
);

CREATE TRIGGER update_identities_updated_at
   BEFORE UPDATE ON "identities"
   FOR EACH ROW EXECUTE PROCEDURE update_updated_at();

CREATE UNIQUE INDEX "identities_idx_1" ON "identities" ("provider" ASC, "subject" ASC);

ALTER TABLE "vehicle_user_access_pending" ADD CONSTRAINT "vehicles_vehicle_user_access_pending"
    FOREIGN KEY ("vehicle_id")
    REFERENCES "vehicles" ("id")
    ON DELETE  CASCADE
    NOT DEFERRABLE
    INITIALLY IMMEDIATE
;

ALTER TABLE "vehicle_user_access" ADD CONSTRAINT "vehicles_vehicle_user_access"
    FOREIGN KEY ("vehicle_id")
    REFERENCES "vehicles" ("id")
    ON DELETE  CASCADE
    NOT DEFERRABLE
    INITIALLY IMMEDIATE
;

ALTER TABLE "vehicles" ADD CONSTRAINT "vehicles_users"
    FOREIGN KEY ("created_by")
    REFERENCES "users" ("id")
    ON DELETE  CASCADE
    NOT DEFERRABLE
    INITIALLY IMMEDIATE
;

ALTER TABLE "maintenance_records" ADD CONSTRAINT "maintenance_records_vehicles"
    FOREIGN KEY ("vehicle_id")
    REFERENCES "vehicles" ("id")
    ON DELETE  CASCADE
    NOT DEFERRABLE
    INITIALLY IMMEDIATE
;

ALTER TABLE "maintenance_records" ADD CONSTRAINT "maintenance_records_users"
    FOREIGN KEY ("created_by")
    REFERENCES "users" ("id")
    NOT DEFERRABLE
    INITIALLY IMMEDIATE
;

ALTER TABLE "vehicle_user_access_pending" ADD CONSTRAINT "vehicle_user_access_pending_users"
    FOREIGN KEY ("user_id")
    REFERENCES "users" ("id")
    ON DELETE  CASCADE
    NOT DEFERRABLE
    INITIALLY IMMEDIATE
;

ALTER TABLE "vehicle_user_access_pending" ADD CONSTRAINT "vehicle_user_access_pending_users_created"
    FOREIGN KEY ("created_by")
    REFERENCES "users" ("id")
    ON DELETE  CASCADE
    NOT DEFERRABLE
    INITIALLY IMMEDIATE
;


ALTER TABLE "vehicle_user_access" ADD CONSTRAINT "vehicle_user_access_users"
    FOREIGN KEY ("user_id")
    REFERENCES "users" ("id")
    ON DELETE  CASCADE
    NOT DEFERRABLE
    INITIALLY IMMEDIATE
;

ALTER TABLE "vehicle_user_access" ADD CONSTRAINT "vehicle_user_access_users_created"
    FOREIGN KEY ("created_by")
    REFERENCES "users" ("id")
    ON DELETE  CASCADE
    NOT DEFERRABLE
    INITIALLY IMMEDIATE
;
