package models

// Product is a structure for storing data for one product from a table.
type Product struct {
	Model    string
	Type     string
	Quantity string
	ImageURL string
	Price    string
}
