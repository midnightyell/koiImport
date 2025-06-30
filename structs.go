package main

import (
	"context"

	koi "gitea.local/smalloy/koiApi"
)

type Item struct {
	koi.Item
	URL             string        `json:"url"`
	EbayID          string        `json:"id"`
	PriceOriginal   string        `json:"price_original"`
	PriceConverted  string        `json:"price_converted"`
	SellerName      string        `json:"seller_name"`
	SellerURL       string        `json:"seller_url"`
	Photos          []string      `json:"photos"`
	DescriptionURL  string        `json:"description_url"`
	DescriptionText string        `json:"description_text"`
	DescriptionHTML string        `json:"description_html"`
	Features        Features      `json:"features"`
	PictureData     []PictureData `json:"picture_data"`
	Skip            bool          `json:"skip"`
	PhotoIndex      int           `json:"photo_index"`
}

type Features map[string]string

type PictureData struct {
	OriginalURLs []string `json:"original_URLs"`
	URL          string   `json:"URL"`
	Ext          string   `json:"ext"`
	Checksum     string   `json:"checksum"`
	Filename     string   `json:"filename"`
	Basename     string   `json:"basename"`
	ItemID       string   `json:"itemid"`
	Resolution   string   `json:"resolution"`
}
type ItemInterface interface {
	koi.ItemInterface
	AddDatum(ctx context.Context, client koi.Client, datumType string, Label string, Value string) (*koi.Datum, error)
}
