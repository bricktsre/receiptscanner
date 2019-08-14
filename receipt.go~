package receiptscanner

type Receipt struct {
	URL string
	ID int64
	Business string
	SubTotal float64
	Tax float64
	Total float64
}

// ReceiptDatabase provides thread-safe access to a database of books.
type ReceiptDatabase interface {
	// GetReceipt retrieves a book by its ID.
	GetReceipt(id int64) (*Receipt, error)

	// AddReceipt saves a given book, assigning it a new ID.
	AddReceipt(b *Receipt) (id int64, err error)

	// DeleteReceipt removes a given book by its ID.
	DeleteReceipt(id int64) error

	// UpdateReceipt updates the entry for a given book.
	UpdateReceipt(b *Receipt) error

	// Close closes the database, freeing up any available resources.
	// TODO(cbro): Close() should return an error.
	Close()
}
