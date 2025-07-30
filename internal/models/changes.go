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

// HasChanges checks if any changes have been detected.
func (c *Changes) HasChanges() bool {
	return len(c.Added) > 0 || len(c.Removed) > 0 || len(c.Changed) > 0
}

// State - the complete state stored in the database.
type State struct {
	PageHash string
	Products []Product
}
