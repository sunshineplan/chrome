package chrome

import (
	"context"

	"github.com/chromedp/cdproto/domstorage"
	"github.com/chromedp/chromedp"
)

func SetStorageItem(ctx context.Context, storageID *domstorage.StorageID, key, value string) error {
	return chromedp.Run(ctx, domstorage.SetDOMStorageItem(storageID, key, value))
}

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
