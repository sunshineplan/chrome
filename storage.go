package chrome

import (
	"context"

	"github.com/chromedp/cdproto/domstorage"
	"github.com/chromedp/chromedp"
)

// SetStorageItem sets a key-value pair in the DOM storage (localStorage/sessionStorage).
func SetStorageItem(ctx context.Context, storageID *domstorage.StorageID, key, value string) error {
	return chromedp.Run(ctx, domstorage.SetDOMStorageItem(storageID, key, value))
}

// StorageItems retrieves all key-value pairs from the DOM storage.
func StorageItems(ctx context.Context, storageID *domstorage.StorageID) (res []domstorage.Item, err error) {
	err = chromedp.Run(
		ctx,
		chromedp.ActionFunc(func(ctx context.Context) (err error) {
			res, err = domstorage.GetDOMStorageItems(storageID).Do(ctx)
			return
		}),
	)
	return
}

// SetStorageItem sets a storage item in this Chrome instance.
func (c *Chrome) SetStorageItem(storageID *domstorage.StorageID, key, value string) error {
	return SetStorageItem(c, storageID, key, value)
}

// StorageItems retrieves storage items from this Chrome instance.
func (c *Chrome) StorageItems(storageID *domstorage.StorageID) ([]domstorage.Item, error) {
	return StorageItems(c, storageID)
}
