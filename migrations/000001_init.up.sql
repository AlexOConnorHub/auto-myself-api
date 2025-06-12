-- Created by Vertabelo (http://vertabelo.com)
-- Last modification date: 2024-11-19 03:31:39.06

-- tables
-- Table: car_maintenance_intervals
CREATE TABLE "car_maintenance_intervals" (
    "id" uuid  NOT NULL,
    "car_id" uuid  NOT NULL,
    "maintenance_type_id" uuid  NOT NULL,
    "miles_between" integer  NULL,
    "weeks_between" integer  NULL,
    "created_by" uuid  NOT NULL,
    "created_at" timestamptz  NOT NULL DEFAULT now(),
    "updated_at" timestamptz  NOT NULL DEFAULT now(),
    "deleted_at" timestamptz,
    CONSTRAINT "car_maintenance_intervals_pk" PRIMARY KEY ("id")
);

CREATE INDEX "car_maintainance_intervals_idx_1" on "car_maintenance_intervals" ("car_id" ASC);

CREATE INDEX "car_maintainance_intervals_idx_2" on "car_maintenance_intervals" ("maintenance_type_id" ASC);

CREATE INDEX "car_maintenance_intervals_idx_3" on "car_maintenance_intervals" ("created_by" ASC);

-- Table: cars
CREATE TABLE "cars" (
    "id" uuid  NOT NULL,
    "make" text  NULL,
    "model" text  NULL,
    "year" integer  NULL,
    "vin" text  NULL,
    "lpn" text  NULL,
    "nickname" text  NULL,
    "annual_mileage" integer  NULL,
    "created_by" uuid  NOT NULL,
    "created_at" timestamptz  NOT NULL DEFAULT now(),
    "updated_at" timestamptz  NOT NULL DEFAULT now(),
    "deleted_at" timestamptz,
    CONSTRAINT "id" PRIMARY KEY ("id")
);

CREATE INDEX "cars_idx_1" on "cars" ("created_by" ASC);

-- Table: deleted
CREATE TABLE "deleted" (
    "id" uuid  NOT NULL,
    "source_table" text  NOT NULL,
    "source_id" uuid  NOT NULL,
    "deleted_at" timestamptz  NOT NULL,
    CONSTRAINT "deleted_pk" PRIMARY KEY ("id")
);

CREATE INDEX "deleted_idx_1" on "deleted" ("source_table" ASC,"source_id" ASC);

-- Table: maintenance_records
CREATE TABLE "maintenance_records" (
    "id" uuid  NOT NULL,
    "odometer" integer  NULL,
    "timestamp" timestamp  NULL,
    "notes" text  NULL,
    "car_id" uuid  NOT NULL,
    "maintenance_type_id" uuid  NOT NULL,
    "created_by" uuid  NOT NULL,
    "created_at" timestamptz  NOT NULL DEFAULT now(),
    "updated_at" timestamptz  NOT NULL DEFAULT now(),
    "deleted_at" timestamptz,
    CONSTRAINT "maintenance_records_pk" PRIMARY KEY ("id")
);

-- Table: maintenance_types
CREATE TABLE "maintenance_types" (
    "id" uuid  NOT NULL,
    "name" text  NOT NULL,
    "created_by" uuid  NOT NULL,
    "created_at" timestamptz  NOT NULL DEFAULT now(),
    "updated_at" timestamptz  NOT NULL DEFAULT now(),
    "deleted_at" timestamptz,
    CONSTRAINT "maintenance_types_pk" PRIMARY KEY ("id")
);

CREATE INDEX "maintenance_types_idx_1" on "maintenance_types" ("created_by" ASC);

-- Table: permissions
CREATE TABLE "permissions" (
    "id" uuid  NOT NULL,
    "user_id" uuid  NOT NULL,
    "car_id" uuid  NOT NULL,
    "write" boolean  NOT NULL,
    "share" boolean  NOT NULL,
    "created_by" uuid  NOT NULL,
    "created_at" timestamptz  NOT NULL DEFAULT now(),
    "updated_at" timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT "permissions_pk" PRIMARY KEY ("id")
);

CREATE INDEX "permissions_idx_1" on "permissions" ("car_id" ASC);

CREATE INDEX "permissions_idx_2" on "permissions" ("user_id" ASC);

CREATE INDEX "permissions_idx_3" on "permissions" ("created_by" ASC);

-- Table: users
CREATE TABLE "users" (
    "id" uuid  NOT NULL,
    "username" text  NULL,
    "created_at" timestamptz  NOT NULL DEFAULT now(),
    "updated_at" timestamptz  NOT NULL DEFAULT now(),
    "deleted_at" timestamptz,
    CONSTRAINT "users_pk" PRIMARY KEY ("id")
);

-- views
-- View: all_permissions
CREATE VIEW "all_permissions" AS
SELECT user_id, car_id, write, share
FROM permissions
UNION ALL
SELECT created_by, id, true, true
FROM cars;

-- foreign keys
-- Reference: car_maintenance_intervals_users (table: car_maintenance_intervals)
ALTER TABLE "car_maintenance_intervals" ADD CONSTRAINT "car_maintenance_intervals_users"
    FOREIGN KEY ("created_by")
    REFERENCES "users" ("id")
    NOT DEFERRABLE
    INITIALLY IMMEDIATE
;

-- Reference: cars_maintainance_types_cars (table: car_maintenance_intervals)
ALTER TABLE "car_maintenance_intervals" ADD CONSTRAINT "cars_maintainance_types_cars"
    FOREIGN KEY ("car_id")
    REFERENCES "cars" ("id")
    ON DELETE  CASCADE
    NOT DEFERRABLE
    INITIALLY IMMEDIATE
;

-- Reference: cars_maintainance_types_maintainance_types (table: car_maintenance_intervals)
ALTER TABLE "car_maintenance_intervals" ADD CONSTRAINT "cars_maintainance_types_maintainance_types"
    FOREIGN KEY ("maintenance_type_id")
    REFERENCES "maintenance_types" ("id")
    ON DELETE  CASCADE
    NOT DEFERRABLE
    INITIALLY IMMEDIATE
;

-- Reference: cars_permissions (table: permissions)
ALTER TABLE "permissions" ADD CONSTRAINT "cars_permissions"
    FOREIGN KEY ("car_id")
    REFERENCES "cars" ("id")
    ON DELETE  CASCADE
    NOT DEFERRABLE
    INITIALLY IMMEDIATE
;

-- Reference: cars_users (table: cars)
ALTER TABLE "cars" ADD CONSTRAINT "cars_users"
    FOREIGN KEY ("created_by")
    REFERENCES "users" ("id")
    ON DELETE  CASCADE
    NOT DEFERRABLE
    INITIALLY IMMEDIATE
;

-- Reference: maintainance_records_cars (table: maintenance_records)
ALTER TABLE "maintenance_records" ADD CONSTRAINT "maintainance_records_cars"
    FOREIGN KEY ("car_id")
    REFERENCES "cars" ("id")
    ON DELETE  CASCADE
    NOT DEFERRABLE
    INITIALLY IMMEDIATE
;

-- Reference: maintainance_records_maintainance_types (table: maintenance_records)
ALTER TABLE "maintenance_records" ADD CONSTRAINT "maintainance_records_maintainance_types"
    FOREIGN KEY ("maintenance_type_id")
    REFERENCES "maintenance_types" ("id")
    NOT DEFERRABLE
    INITIALLY IMMEDIATE
;

-- Reference: maintenance_records_users (table: maintenance_records)
ALTER TABLE "maintenance_records" ADD CONSTRAINT "maintenance_records_users"
    FOREIGN KEY ("created_by")
    REFERENCES "users" ("id")
    NOT DEFERRABLE
    INITIALLY IMMEDIATE
;

-- Reference: maintenance_types_users (table: maintenance_types)
ALTER TABLE "maintenance_types" ADD CONSTRAINT "maintenance_types_users"
    FOREIGN KEY ("created_by")
    REFERENCES "users" ("id")
    NOT DEFERRABLE
    INITIALLY IMMEDIATE
;

-- Reference: permissions_users (table: permissions)
ALTER TABLE "permissions" ADD CONSTRAINT "permissions_users"
    FOREIGN KEY ("user_id")
    REFERENCES "users" ("id")
    ON DELETE  CASCADE
    NOT DEFERRABLE
    INITIALLY IMMEDIATE
;

-- Reference: permissions_users_created (table: permissions)
ALTER TABLE "permissions" ADD CONSTRAINT "permissions_users_created"
    FOREIGN KEY ("created_by")
    REFERENCES "users" ("id")
    ON DELETE  CASCADE
    NOT DEFERRABLE
    INITIALLY IMMEDIATE
;

-- End of file.
