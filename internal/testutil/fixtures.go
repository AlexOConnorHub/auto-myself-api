package testutil

import "github.com/gofrs/uuid/v5"

const (
	NO_ACCESS = iota
	READ_BY_INVITE
	READ_ONLY
	WRITE
)

type UserFixture struct {
	ID       uuid.UUID
	Username string
	Role     string
}

type VehicleFixture struct {
	ID       uuid.UUID
	Nickname string
	Year     int
	Make     string
	MakeID   int
	Model    string
	ModelID  int
	Vin      string
	Lpn      string
	OwnerID  uuid.UUID
}

type ShareFixture struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	VehicleID   uuid.UUID
	WriteAccess bool
	CreatedBy   uuid.UUID
}

type MaintenanceFixture struct {
	ID           uuid.UUID
	VehicleID    uuid.UUID
	CreatedBy    uuid.UUID
	Odometer     int
	Timestamp    string
	Notes        string
	Type         string
	Interval     int
	IntervalType string
	Cost         string
}

var (
	User1ID = mustUUID("019785fe-4eb4-766e-9c45-bec7780972a2")
	User2ID = mustUUID("019785fe-4eb4-766e-9c45-c1f83e7c1f1f")
	User3ID = mustUUID("019785fe-4eb4-766e-9c45-c497f2d9fe9e")
	User4ID = mustUUID("019785fe-4eb4-766e-9c45-c8578456b4df")
	User5ID = mustUUID("019785fe-4eb4-766e-9c45-cec136a9ad6f")
	User6ID = mustUUID("019785fe-4eb4-766e-9c45-f592a1187d0c")
	User7ID = mustUUID("019785fe-4eb4-766e-9c45-f9cd4ee5c0b3")
	User8ID = mustUUID("019785fe-4eb4-766e-9c45-fc6ed4a7407b")

	Vehicle1ID = mustUUID("019785fe-4eb4-766e-9c45-d0b2bb289b82")
	Vehicle2ID = mustUUID("019785fe-4eb4-766e-9c45-d77f41aa8317")
	Vehicle3ID = mustUUID("019785fe-4eb4-766e-9c45-d9cc7ea628c1")
	Vehicle4ID = mustUUID("019785fe-4eb4-766e-9c45-ddfb4b2e7210")
	Vehicle5ID = mustUUID("019785fe-4eb4-766e-9c45-e1af5010246b")

	Share1ID = mustUUID("01988201-ec4e-7bae-88eb-17fd5bf60ca8")
	Share2ID = mustUUID("019d5352-0a33-78e5-9148-59bac3670a1a")
)

var Users = []UserFixture{
	{ID: User1ID, Username: "User 1", Role: "admin"},
	{ID: User2ID, Username: "User 2", Role: "moderator"},
	{ID: User3ID, Username: "User 3", Role: "user"},
	{ID: User4ID, Username: "User 4", Role: "user"},
	{ID: User5ID, Username: "User 5", Role: "user"},
	{ID: User6ID, Username: "User 6", Role: "user"},
	{ID: User7ID, Username: "User 7", Role: "user"},
	{ID: User8ID, Username: "User 8", Role: "user"},
}

var UserIDs = []uuid.UUID{
	User1ID, User2ID, User3ID, User4ID, User5ID, User6ID, User7ID, User8ID,
}

var Vehicles = []VehicleFixture{
	{ID: Vehicle1ID, Nickname: "Vehicle 1", Year: 2015, Make: "MAZDA", MakeID: 473, Model: "MX-5", ModelID: 2072, Vin: "VIN CAR 1", Lpn: "LPN CAR 1", OwnerID: User1ID},
	{ID: Vehicle2ID, Nickname: "Vehicle 2", Year: 2021, Make: "TOYOTA", MakeID: 448, Model: "Corolla", ModelID: 2208, Vin: "VIN CAR 2", Lpn: "LPN CAR 2", OwnerID: User1ID},
	{ID: Vehicle3ID, Nickname: "Vehicle 3", Year: 2019, Make: "FORD", MakeID: 460, Model: "Edge", ModelID: 1797, Vin: "VIN CAR 3", Lpn: "LPN CAR 3", OwnerID: User4ID},
	{ID: Vehicle4ID, Nickname: "Vehicle 4", Year: 2018, Make: "NISSAN", MakeID: 478, Model: "GT-R", ModelID: 1890, Vin: "VIN CAR 4", Lpn: "LPN CAR 4", OwnerID: User6ID},
	{ID: Vehicle5ID, Nickname: "Vehicle 5", Year: 2025, Make: "ASTON MARTIN", MakeID: 440, Model: "Vanquish", ModelID: 1701, Vin: "VIN CAR 5", Lpn: "LPN CAR 5", OwnerID: User8ID},
}

var VehicleIDs = []uuid.UUID{Vehicle1ID, Vehicle2ID, Vehicle3ID, Vehicle4ID, Vehicle5ID}

var Shares = []ShareFixture{
	{ID: Share1ID, UserID: User5ID, VehicleID: Vehicle4ID, WriteAccess: true, CreatedBy: User6ID},
	{ID: Share2ID, UserID: User4ID, VehicleID: Vehicle4ID, WriteAccess: false, CreatedBy: User6ID},
}

var Maintenances = []MaintenanceFixture{
	{ID: mustUUID("01978640-1148-74f8-be64-59f2af568e59"), VehicleID: Vehicle1ID, CreatedBy: User1ID, Odometer: 15000, Timestamp: "2023-01-15", Notes: "Oil change", Type: "Oil Change", Interval: 5000, IntervalType: "miles", Cost: "$1.32"},
	{ID: mustUUID("01978640-1148-74f8-be64-5e6b15475861"), VehicleID: Vehicle1ID, CreatedBy: User1ID, Odometer: 20000, Timestamp: "2023-06-20", Notes: "Tire rotation", Type: "Tire Rotation", Interval: 10000, IntervalType: "miles", Cost: "$1.32"},
	{ID: mustUUID("01978640-1148-74f8-be64-600b58c80190"), VehicleID: Vehicle2ID, CreatedBy: User1ID, Odometer: 5000, Timestamp: "2023-02-10", Notes: "Brake inspection", Type: "Inspection", Interval: 10000, IntervalType: "miles", Cost: "$1.32"},
	{ID: mustUUID("01978640-1148-74f8-be64-673c2bc659d3"), VehicleID: Vehicle2ID, CreatedBy: User1ID, Odometer: 12000, Timestamp: "2023-07-05", Notes: "Battery check", Type: "Maintenance Check", Interval: 20000, IntervalType: "miles", Cost: "$1.32"},
	{ID: mustUUID("01978640-1148-74f8-be64-6b2a85c627a7"), VehicleID: Vehicle2ID, CreatedBy: User2ID, Odometer: 8000, Timestamp: "2023-03-12", Notes: "Fluid top-up", Type: "Maintenance Check", Interval: 5000, IntervalType: "miles", Cost: "$1.32"},
	{ID: mustUUID("01978640-1148-74f8-be64-6ce4acd8abcd"), VehicleID: Vehicle2ID, CreatedBy: User2ID, Odometer: 16000, Timestamp: "2023-08-18", Notes: "Tire replacement", Type: "Tire Replacement", Interval: 20000, IntervalType: "miles", Cost: "$1.32"},
	{ID: mustUUID("01978640-1148-74f8-be64-70821e946a20"), VehicleID: Vehicle2ID, CreatedBy: User7ID, Odometer: 3000, Timestamp: "2023-04-05", Notes: "Oil change", Type: "Oil Change", Interval: 5000, IntervalType: "miles", Cost: "$1.32"},
	{ID: mustUUID("01978640-1148-74f8-be64-74b319513577"), VehicleID: Vehicle2ID, CreatedBy: User7ID, Odometer: 7000, Timestamp: "2023-09-10", Notes: "Brake pad replacement", Type: "Brake Replacement", Interval: 10000, IntervalType: "miles", Cost: "$1.32"},
	{ID: mustUUID("01978640-1148-74f8-be64-7b8ece4dc40f"), VehicleID: Vehicle2ID, CreatedBy: User8ID, Odometer: 1000, Timestamp: "2023-05-15", Notes: "Inspection check", Type: "Inspection Check", Interval: 5000, IntervalType: "miles", Cost: "$1.32"},
	{ID: mustUUID("01978640-1148-74f8-be64-7ea0e41d68d5"), VehicleID: Vehicle2ID, CreatedBy: User8ID, Odometer: 4000, Timestamp: "2023-10-20", Notes: "Tire rotation and balance", Type: "Tire Rotation and Balance", Interval: 10000, IntervalType: "miles", Cost: "$1.32"},
	{ID: mustUUID("01978640-1148-74f8-be64-bc7c09adc4c1"), VehicleID: Vehicle3ID, CreatedBy: User4ID, Odometer: 3000, Timestamp: "2023-04-05", Notes: "Oil change", Type: "Oil Change", Interval: 5000, IntervalType: "miles", Cost: "$1.32"},
	{ID: mustUUID("01978640-1148-74f8-be64-ac02bbbf11cf"), VehicleID: Vehicle3ID, CreatedBy: User4ID, Odometer: 7000, Timestamp: "2023-09-10", Notes: "Brake pad replacement", Type: "Brake Replacement", Interval: 10000, IntervalType: "miles", Cost: "$1.32"},
	{ID: mustUUID("01978640-1148-74f8-be64-b446e827f938"), VehicleID: Vehicle4ID, CreatedBy: User6ID, Odometer: 3000, Timestamp: "2023-04-05", Notes: "Oil change", Type: "Oil Change", Interval: 5000, IntervalType: "miles", Cost: "$1.32"},
	{ID: mustUUID("01978640-1148-74f8-be64-ba806fa103c7"), VehicleID: Vehicle4ID, CreatedBy: User6ID, Odometer: 7000, Timestamp: "2023-09-10", Notes: "Brake pad replacement", Type: "Brake Replacement", Interval: 10000, IntervalType: "miles", Cost: "$1.32"},
	{ID: mustUUID("01978640-1149-7118-bada-9f77b4fa870a"), VehicleID: Vehicle4ID, CreatedBy: User7ID, Odometer: 2000, Timestamp: "2023-03-01", Notes: "Oil change", Type: "Oil Change", Interval: 5000, IntervalType: "miles", Cost: "$1.32"},
	{ID: mustUUID("01978640-1149-7118-bada-a16e466c1064"), VehicleID: Vehicle4ID, CreatedBy: User7ID, Odometer: 4000, Timestamp: "2023-08-15", Notes: "Tire rotation and balance", Type: "Tire Rotation and Balance", Interval: 10000, IntervalType: "miles", Cost: "$1.32"},
	{ID: mustUUID("01978640-1149-7118-bada-a7ce46886414"), VehicleID: Vehicle5ID, CreatedBy: User8ID, Odometer: 5000, Timestamp: "2023-02-10", Notes: "Brake inspection", Type: "Inspection", Interval: 10000, IntervalType: "miles", Cost: "$1.32"},
	{ID: mustUUID("01978640-1149-7118-bada-aa596985d112"), VehicleID: Vehicle5ID, CreatedBy: User8ID, Odometer: 12000, Timestamp: "2023-07-05", Notes: "Battery check", Type: "Maintenance Check", Interval: 20000, IntervalType: "miles", Cost: "$1.32"},
}

var MaintenanceIDs = func() []uuid.UUID {
	ids := make([]uuid.UUID, len(Maintenances))
	for i, record := range Maintenances {
		ids[i] = record.ID
	}
	return ids
}()

var VehicleAccessMatrix = [8][5]int{
	{WRITE, WRITE, NO_ACCESS, NO_ACCESS, NO_ACCESS},
	{NO_ACCESS, WRITE, NO_ACCESS, NO_ACCESS, NO_ACCESS},
	{NO_ACCESS, READ_ONLY, NO_ACCESS, NO_ACCESS, NO_ACCESS},
	{NO_ACCESS, NO_ACCESS, WRITE, READ_BY_INVITE, NO_ACCESS},
	{NO_ACCESS, NO_ACCESS, NO_ACCESS, READ_BY_INVITE, NO_ACCESS},
	{NO_ACCESS, NO_ACCESS, NO_ACCESS, WRITE, NO_ACCESS},
	{NO_ACCESS, WRITE, NO_ACCESS, WRITE, NO_ACCESS},
	{NO_ACCESS, WRITE, NO_ACCESS, READ_ONLY, WRITE},
}

var UserAccessMatrix = [8][8]int{
	{WRITE, READ_ONLY, READ_ONLY, NO_ACCESS, NO_ACCESS, NO_ACCESS, READ_ONLY, READ_ONLY},
	{READ_ONLY, WRITE, NO_ACCESS, NO_ACCESS, NO_ACCESS, NO_ACCESS, READ_ONLY, READ_ONLY},
	{READ_ONLY, READ_ONLY, WRITE, NO_ACCESS, NO_ACCESS, NO_ACCESS, READ_ONLY, READ_ONLY},
	{NO_ACCESS, NO_ACCESS, NO_ACCESS, WRITE, NO_ACCESS, NO_ACCESS, NO_ACCESS, NO_ACCESS},
	{NO_ACCESS, NO_ACCESS, NO_ACCESS, NO_ACCESS, WRITE, NO_ACCESS, NO_ACCESS, NO_ACCESS},
	{NO_ACCESS, NO_ACCESS, NO_ACCESS, NO_ACCESS, NO_ACCESS, WRITE, READ_ONLY, READ_ONLY},
	{READ_ONLY, READ_ONLY, NO_ACCESS, NO_ACCESS, NO_ACCESS, READ_ONLY, WRITE, READ_ONLY},
	{READ_ONLY, READ_ONLY, NO_ACCESS, NO_ACCESS, NO_ACCESS, READ_ONLY, READ_ONLY, WRITE},
}

var ShareAccessMatrix = [8][2]int{
	{NO_ACCESS, NO_ACCESS},
	{NO_ACCESS, NO_ACCESS},
	{NO_ACCESS, NO_ACCESS},
	{NO_ACCESS, WRITE},
	{WRITE, NO_ACCESS},
	{READ_ONLY, READ_ONLY},
	{NO_ACCESS, NO_ACCESS},
	{NO_ACCESS, NO_ACCESS},
}

func mustUUID(value string) uuid.UUID {
	id, err := uuid.FromString(value)
	if err != nil {
		panic("invalid test fixture UUID " + value + ": " + err.Error())
	}
	return id
}
