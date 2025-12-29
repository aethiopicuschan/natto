package stun

// This file exposes unexported functions for black-box tests
// in package stun_test. It is compiled only during `go test`.

// stunType / parseType
var TestStunType = stunType
var TestParseType = parseType

// endian helpers
var TestReadU16 = readU16
var TestReadU32 = readU32
var TestPutU16 = putU16
var TestPutU32 = putU32
