package models

// ChangeInfo - information about the changed product.
type ChangeInfo struct {
	Old Product
	New Product
}

// Changes - comparison result: all types of changes.
type Changes struct {
	Added   []Product
	Removed []Product
	Changed []ChangeInfo
}

// State - the complete state stored in the database.
type State struct {
	PageHash string
	Products []Product
}
