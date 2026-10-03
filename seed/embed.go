// Package seeddata embeds the seed files the server needs at run time (not only when seeding).
package seeddata

import _ "embed"

// Geo is the divisions, districts and upazilas /geo answers.
//
//go:embed geo.json
var Geo []byte
