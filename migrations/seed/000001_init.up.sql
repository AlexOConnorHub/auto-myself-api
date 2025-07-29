INSERT INTO "users" ("id", "username") VALUES
('019785fe-4eb4-766e-9c45-bec7780972a2', 'User 1'), -- Main test user: One personal vehicle, one vehicle to share
('019785fe-4eb4-766e-9c45-c1f83e7c1f1f', 'User 2'), -- Vehicle shared from user1 with write access
('019785fe-4eb4-766e-9c45-c497f2d9fe9e', 'User 3'), -- Vehicle shared from user1 with read access
('019785fe-4eb4-766e-9c45-c8578456b4df', 'User 4'), -- Has one personal vehicle
('019785fe-4eb4-766e-9c45-cec136a9ad6f', 'User 5'), -- Has no vehicles, no vehicle shared
('019785fe-4eb4-766e-9c45-f592a1187d0c', 'User 6'), -- Has vehicle to share
('019785fe-4eb4-766e-9c45-f9cd4ee5c0b3', 'User 7'), -- Vehicle shared from user1 and user6, both write
('019785fe-4eb4-766e-9c45-fc6ed4a7407b', 'User 8'); -- Has personal vehicle, vehicle shared from user1 (write) and user6 (read)

INSERT INTO "vehicles" ("id", "nickname", "year", "make", "make_id", "model", "model_id", "vin", "lpn", "created_by") VALUES
('019785fe-4eb4-766e-9c45-d0b2bb289b82', 'Vehicle 1', 2015, 'MAZDA', 473, 'MX-5', 2072, 'VIN CAR 1', 'LPN CAR 1', '019785fe-4eb4-766e-9c45-bec7780972a2'), -- User 1 personal vehicle
('019785fe-4eb4-766e-9c45-d77f41aa8317', 'Vehicle 2', 2021, 'TOYOTA', 448, 'Corolla', 2208, 'VIN CAR 2', 'LPN CAR 2', '019785fe-4eb4-766e-9c45-bec7780972a2'), -- User 1 vehicle to share
('019785fe-4eb4-766e-9c45-d9cc7ea628c1', 'Vehicle 3', 2019, 'FORD', 460, 'Edge', 1797, 'VIN CAR 3', 'LPN CAR 3', '019785fe-4eb4-766e-9c45-c8578456b4df'), -- User 4 personal vehicle, not shared
('019785fe-4eb4-766e-9c45-ddfb4b2e7210', 'Vehicle 4', 2018, 'NISSAN', 478, 'GT-R', 1890, 'VIN CAR 4', 'LPN CAR 4', '019785fe-4eb4-766e-9c45-f592a1187d0c'), -- User 6 vehicle to share
('019785fe-4eb4-766e-9c45-e1af5010246b', 'Vehicle 5', 2025, 'ASTON MARTIN', 440, 'Vanquish', 1701, 'VIN CAR 5', 'LPN CAR 5', '019785fe-4eb4-766e-9c45-fc6ed4a7407b'); -- User 8 personal vehicle

INSERT INTO "vehicle_user_access" ("id", "user_id", "vehicle_id", "write_access", "created_by") VALUES
('019785fe-4eb4-766e-9c45-e4287c35fe36', '019785fe-4eb4-766e-9c45-c1f83e7c1f1f', '019785fe-4eb4-766e-9c45-d77f41aa8317', true, '019785fe-4eb4-766e-9c45-bec7780972a2'), -- User 2 access to Vehicle 2 (write)
('019785fe-4eb4-766e-9c45-ea1fef642df3', '019785fe-4eb4-766e-9c45-c497f2d9fe9e', '019785fe-4eb4-766e-9c45-d77f41aa8317', false, '019785fe-4eb4-766e-9c45-bec7780972a2'), -- User 3 access to Vehicle 2 (read)
('019785fe-4eb4-766e-9c45-ed47aa22a565', '019785fe-4eb4-766e-9c45-f9cd4ee5c0b3', '019785fe-4eb4-766e-9c45-d77f41aa8317', true, '019785fe-4eb4-766e-9c45-bec7780972a2'), -- User 7 access to Vehicle 2 (write)
('019785fe-4eb4-766e-9c45-f0684e1dcd13', '019785fe-4eb4-766e-9c45-f9cd4ee5c0b3', '019785fe-4eb4-766e-9c45-ddfb4b2e7210', true, '019785fe-4eb4-766e-9c45-f592a1187d0c'), -- User 7 access to Vehicle 4 (write)
('019785fe-4eb4-766e-9c46-021143f69370', '019785fe-4eb4-766e-9c45-fc6ed4a7407b', '019785fe-4eb4-766e-9c45-d77f41aa8317', true, '019785fe-4eb4-766e-9c45-bec7780972a2'), -- User 8 access to Vehicle 2 (write)
('019785fe-4eb4-766e-9c46-045710f92920', '019785fe-4eb4-766e-9c45-fc6ed4a7407b', '019785fe-4eb4-766e-9c45-ddfb4b2e7210', false, '019785fe-4eb4-766e-9c45-f592a1187d0c'); -- User 8 access to Vehicle 4 (read)

INSERT INTO "maintenance_records" ("id", "vehicle_id", "odometer", "timestamp", "notes", "type", "interval", "interval_type", "created_by", "cost") VALUES -- Add 2 maintenance records for every user that can write them to vehicles
('01978640-1148-74f8-be64-59f2af568e59', '019785fe-4eb4-766e-9c45-d0b2bb289b82', 15000, '2023-01-15', 'Oil change', 'Oil Change', 5000, 'miles', '019785fe-4eb4-766e-9c45-bec7780972a2', '$1.32'), -- User 1 Vehicle 1
('01978640-1148-74f8-be64-5e6b15475861', '019785fe-4eb4-766e-9c45-d0b2bb289b82', 20000, '2023-06-20', 'Tire rotation', 'Tire Rotation', 10000, 'miles', '019785fe-4eb4-766e-9c45-bec7780972a2', '$1.32'), -- User 1 Vehicle 1
('01978640-1148-74f8-be64-600b58c80190', '019785fe-4eb4-766e-9c45-d77f41aa8317', 5000, '2023-02-10', 'Brake inspection', 'Inspection', 10000, 'miles', '019785fe-4eb4-766e-9c45-bec7780972a2', '$1.32'), -- User 1 Vehicle 2
('01978640-1148-74f8-be64-673c2bc659d3', '019785fe-4eb4-766e-9c45-d77f41aa8317', 12000, '2023-07-05', 'Battery check', 'Maintenance Check', 20000, 'miles', '019785fe-4eb4-766e-9c45-bec7780972a2', '$1.32'), -- User 1 Vehicle 2
('01978640-1148-74f8-be64-6b2a85c627a7', '019785fe-4eb4-766e-9c45-d77f41aa8317', 8000, '2023-03-12', 'Fluid top-up', 'Maintenance Check', 5000, 'miles', '019785fe-4eb4-766e-9c45-c1f83e7c1f1f', '$1.32'), -- User 2 Vehicle 2
('01978640-1148-74f8-be64-6ce4acd8abcd', '019785fe-4eb4-766e-9c45-d77f41aa8317', 16000, '2023-08-18', 'Tire replacement', 'Tire Replacement', 20000, 'miles', '019785fe-4eb4-766e-9c45-c1f83e7c1f1f', '$1.32'), -- User 2 Vehicle 2
('01978640-1148-74f8-be64-70821e946a20', '019785fe-4eb4-766e-9c45-d77f41aa8317', 3000, '2023-04-05', 'Oil change', 'Oil Change', 5000, 'miles', '019785fe-4eb4-766e-9c45-f9cd4ee5c0b3', '$1.32'), -- User 7 Vehicle 2
('01978640-1148-74f8-be64-74b319513577', '019785fe-4eb4-766e-9c45-d77f41aa8317', 7000, '2023-09-10', 'Brake pad replacement', 'Brake Replacement', 10000, 'miles', '019785fe-4eb4-766e-9c45-f9cd4ee5c0b3', '$1.32'), -- User 7 Vehicle 2
('01978640-1148-74f8-be64-7b8ece4dc40f', '019785fe-4eb4-766e-9c45-d77f41aa8317', 1000, '2023-05-15', 'Inspection check', 'Inspection Check', 5000, 'miles', '019785fe-4eb4-766e-9c45-fc6ed4a7407b', '$1.32'), -- User 8 Vehicle 2
('01978640-1148-74f8-be64-7ea0e41d68d5', '019785fe-4eb4-766e-9c45-d77f41aa8317', 4000, '2023-10-20', 'Tire rotation and balance', 'Tire Rotation and Balance', 10000, 'miles', '019785fe-4eb4-766e-9c45-fc6ed4a7407b', '$1.32'), -- User 8 Vehicle 2
('01978640-1148-74f8-be64-bc7c09adc4c1', '019785fe-4eb4-766e-9c45-d9cc7ea628c1', 3000, '2023-04-05', 'Oil change', 'Oil Change', 5000, 'miles', '019785fe-4eb4-766e-9c45-c8578456b4df', '$1.32'), -- User 4 Vehicle 3
('01978640-1148-74f8-be64-ac02bbbf11cf', '019785fe-4eb4-766e-9c45-d9cc7ea628c1', 7000, '2023-09-10', 'Brake pad replacement', 'Brake Replacement', 10000, 'miles', '019785fe-4eb4-766e-9c45-c8578456b4df', '$1.32'), -- User 4 Vehicle 3
('01978640-1148-74f8-be64-b446e827f938', '019785fe-4eb4-766e-9c45-ddfb4b2e7210', 3000, '2023-04-05', 'Oil change', 'Oil Change', 5000, 'miles', '019785fe-4eb4-766e-9c45-f592a1187d0c', '$1.32'), -- User 6 Vehicle 4
('01978640-1148-74f8-be64-ba806fa103c7', '019785fe-4eb4-766e-9c45-ddfb4b2e7210', 7000, '2023-09-10', 'Brake pad replacement', 'Brake Replacement', 10000, 'miles', '019785fe-4eb4-766e-9c45-f592a1187d0c', '$1.32'), -- User 6 Vehicle 4
('01978640-1149-7118-bada-9f77b4fa870a', '019785fe-4eb4-766e-9c45-ddfb4b2e7210', 2000, '2023-03-01', 'Oil change', 'Oil Change', 5000, 'miles', '019785fe-4eb4-766e-9c45-f9cd4ee5c0b3', '$1.32'), -- User 7 Vehicle 4
('01978640-1149-7118-bada-a16e466c1064', '019785fe-4eb4-766e-9c45-ddfb4b2e7210', 4000, '2023-08-15', 'Tire rotation and balance', 'Tire Rotation and Balance', 10000, 'miles', '019785fe-4eb4-766e-9c45-f9cd4ee5c0b3', '$1.32'), -- User 7 Vehicle 4
('01978640-1149-7118-bada-a7ce46886414', '019785fe-4eb4-766e-9c45-e1af5010246b', 5000, '2023-02-10', 'Brake inspection', 'Inspection', 10000, 'miles', '019785fe-4eb4-766e-9c45-fc6ed4a7407b', '$1.32'), -- User 8 Vehicle 5
('01978640-1149-7118-bada-aa596985d112', '019785fe-4eb4-766e-9c45-e1af5010246b', 12000, '2023-07-05', 'Battery check', 'Maintenance Check', 20000, 'miles', '019785fe-4eb4-766e-9c45-fc6ed4a7407b', '$1.32'); -- User 8 Vehicle 5

INSERT INTO "deleted" ("id", "source_table", "source_id", "deleted_at") VALUES
('01978666-50ab-7399-9d85-7e7ef5978088', 'users', '01978666-50ab-7399-9d85-8dfbc2857624', '2023-10-01 12:00:00'),
('01978666-50ab-7399-9d85-83f6ae1ef88f', 'vehicles', '01978666-50ab-7399-9d85-905063f4cc7e', '2023-10-01 12:00:00'),
('01978666-50ab-7399-9d85-85cb45aef7b5', 'maintenance_records', '01978666-50ab-7399-9d85-94b7139f0a65', '2023-10-01 12:00:00'),
('01978666-50ab-7399-9d85-89a3a492a139', 'vehicle_user_access', '01978666-50ab-7399-9d85-999ba98b049d', '2023-10-01 12:00:00');

INSERT INTO "users" ("id", "username", "deleted_at") VALUES
('01978666-50ab-7399-9d85-9d4479bda3fd', 'deleted_user1', '2023-10-01 12:00:00');

INSERT INTO "vehicles" ("id", "nickname", "created_by", "deleted_at") VALUES
('01978666-50ab-7399-9d85-aef68bddfd22', 'deleted_vehicle1', '01978666-50ab-7399-9d85-9d4479bda3fd', '2023-10-01 12:00:00');

INSERT INTO "vehicle_user_access" ("id", "user_id", "vehicle_id", "created_by", "deleted_at") VALUES
('01978666-50ab-7399-9d85-b3e259e67262', '01978666-50ab-7399-9d85-9d4479bda3fd', '01978666-50ab-7399-9d85-aef68bddfd22', '01978666-50ab-7399-9d85-9d4479bda3fd', '2023-10-01 12:00:00');

INSERT INTO "maintenance_records" ("id", "vehicle_id", "created_by", "deleted_at") VALUES
('01978666-50ab-7399-9d85-b4852d6c40ae', '01978666-50ab-7399-9d85-aef68bddfd22', '01978666-50ab-7399-9d85-9d4479bda3fd', '2023-10-01 12:00:00');

