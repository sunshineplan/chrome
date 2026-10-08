package chrome

import (
	"context"

	"github.com/chromedp/cdproto/domstorage"
	"github.com/chromedp/chromedp"
)

// SetStorageItem sets a key-value pair in the DOM storage (localStorage/sessionStorage).
func SetStorageItem(ctx context.Context, storageID *domstorage.StorageID, key, value string) (err error) {
	_, err = chromedp.Call(
		ctx,
		domstorage.SetDOMStorageItem,
		domstorage.SetDOMStorageItemParams{
			StorageID: storageID,
			Key:       key,
			Value:     value,
		},
	)
	return
}

// StorageItems retrieves all key-value pairs from the DOM storage.
func StorageItems(ctx context.Context, storageID *domstorage.StorageID) ([]domstorage.Item, error) {
	res, err := chromedp.Call(
		ctx,
		domstorage.GetDOMStorageItems,
		domstorage.GetDOMStorageItemsParams{StorageID: storageID},
	)
	if err != nil {
		return nil, err
	}
	return res.Entries, nil
}

// SetStorageItem sets a storage item in this Chrome instance.
func (c *Chrome) SetStorageItem(storageID *domstorage.StorageID, key, value string) error {
	return SetStorageItem(c, storageID, key, value)
}

// StorageItems retrieves storage items from this Chrome instance.
func (c *Chrome) StorageItems(storageID *domstorage.StorageID) ([]domstorage.Item, error) {
	return StorageItems(c, storageID)
}
