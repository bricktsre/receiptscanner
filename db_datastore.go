package receiptscanner

import (
	"context"
	"fmt"
	
	"cloud.google.com/go/datastore"
)

type datastoreDB struct {
	client *datastore.Client
}

var _RecieptDatabase = &datastoreDB{}

// Creates a new RecieptDatabase backed by Cloud Datastore
func newDatastoreDB(client *datastore.Client)(ReceiptDatabase, error){
	ctx := context.Background()
	// Verify that we can communicate and authenticate with the datastore service.
	t, err := client.NewTransaction(ctx)
	if err != nil {
		return nil, fmt.Errorf("datastoredb: could not connect: %v", err)
	}
	if err := t.Rollback(); err != nil {
		return nil, fmt.Errorf("datastoredb: could not connect: %v", err)
	}
	return &datastoreDB{
		client: client,
	}, nil
}

// Closes the database
func (db *datastoreDB) Close() { }

func (db *datastoreDB) datastoreKey(id int64) *datastore.Key {
	return datastore.IDKey("Receipt", id, nil)
}

// GetReceipt retrieves a receipt by its ID.
func (db *datastoreDB) GetReceipt(id int64) (*Receipt, error) {
	ctx := context.Background()
	k := db.datastoreKey(id)
	receipt := &Receipt{}
	if err := db.client.Get(ctx, k, receipt); err != nil {
		return nil, fmt.Errorf("datastoredb: could not get receipt: %v", err)
	}
	receipt.ID = id
	return receipt, nil
}

// AddReciept saves a given receipt, assigning it a new ID
func (db *datastoreDB) AddReceipt(r *Receipt) (id int64, err error) {
	ctx := context.Background()
	k := datastore.IncompleteKey("Receipt", nil)
	k, err = db.client.Put(ctx, k, r)
	if err != nil {
		return 0, fmt.Errorf("datastoredb: could not put receipt: %v", err)
	}
	return k.ID, nil
}

// DeleteReceipt removes a given Receipt by its ID.
func (db *datastoreDB) DeleteReceipt(id int64) error {
	ctx := context.Background()
	k := db.datastoreKey(id)
	if err := db.client.Delete(ctx, k); err != nil {
		return fmt.Errorf("datastoredb: could not delete Receipt: %v", err)
	}
	return nil
}

// UpdateReceipt updates the entry for a given Receipt.
func (db *datastoreDB) UpdateReceipt(r *Receipt) error {
	ctx := context.Background()
	k := db.datastoreKey(r.ID)
	if _, err := db.client.Put(ctx, k, r); err != nil {
		return fmt.Errorf("datastoredb: could not update Receipt: %v", err)
	}
	return nil
}
