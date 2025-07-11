package dataembed

import _ "embed"

//go:embed data.json
var DataFileBytes []byte
