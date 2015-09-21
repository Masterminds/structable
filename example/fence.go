package main

import (
	"github.com/Masterminds/squirrel"
	"github.com/Masterminds/structable"
)

const FenceTable = "fences"

// Fence represents a Geofence boundary.
//
// This struct is stubbed out to show how an ActiveRecord pattern might look
// implemented using Squirrel and Structable.
//
// The DDL for the underlying table may look something like this:
//  CREATE TABLE fences (
//  id          SERIAL,
//  radius      NUMERIC(20, 14),
//  latitude    NUMERIC(20, 14),
//  longitude   NUMERIC(20, 14),
//  region      INTEGER,
//
//  PRIMARY KEY(id),
//  );
type Fence struct {
	Id        int     `stbl:"id,PRIMARY_KEY,SERIAL"`
	Region    int     `stbl:"region"`
	Radius    float64 `stbl:"radius"`
	Latitude  float64 `stbl:"latitude"`
	Longitude float64 `stbl:"longitude"`

	rec     structable.Recorder
	builder squirrel.StatementBuilderType
}

// NewFence creates a new empty fence.
//
// Note that a DBProxy is Squirrel's interface
// that describes most sql.DB-like things.
//
// Flavor may be one of 'mysql', 'postgres'. Other DBs may
// work, but are untested.
func NewFence(db squirrel.DBProxyBeginner, dbFlavor string) *Fence {
	f := new(Fence)
	f.builder = squirrel.StatementBuilder.RunWith(db)

	// For Postgres we convert '?' to '$N' placeholders.
	if dbFlavor == "postgres" {
		f.builder = f.builder.PlaceholderFormat(squirrel.Dollar)
	}

	f.rec = structable.New(db, dbFlavor).Bind(FenceTable, f)

	return f
}

// Insert creates a new record.
func (r *Fence) Insert() error {
	return r.rec.Insert()
}

// Update modifies an existing record
func (r *Fence) Update() error {
	return r.rec.Update()
}

// Delete removes a record.
func (r *Fence) Delete() error {
	return r.rec.Delete()
}

// Has returns true if the record exists.
func (r *Fence) Has() (bool, error) {
	return r.rec.Exists()
}

// Load populates the struct with data from storage.
// It presumes that the id field is set.
func (r *Fence) Load() error {
	return r.rec.Load()
}

// LoadGeopoint loads by a given Lat/Long
// Example of a custom loader
//
// Usage:
//  fence := NewFence(myDb, "postgres")
//  fence.Latitude = 1.000001
//  fence.Longitude = 1.000002
//  if err := fence.LoadGeopoint(); err != nil {
//    panic("Something went wrong! " + err.Error())
//  }
//  fmt.Printf("Loaded ID %d\n", fence.Id)
//
func (r *Fence) LoadGeopoint() error {
	//q := r.rec.Select("id, radius, region").From(FenceTable).
	//  Where("latitude = ? AND longitude = ?", r.Latitude, r.Longitude)

	//return q.Query().Scan(&r.Id, &r.Radius, &r.Region)
	return r.rec.LoadWhere("latitude = ? AND longitude = ?", r.Latitude, r.Longitude)
}
