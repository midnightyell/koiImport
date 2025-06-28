package main

import (
	"context"
	"fmt"
	"reflect"

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

func structToKVold(v interface{}) ([]struct {
	Key   string
	Value interface{}
}, error) {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("input must be a struct or pointer to a struct")
	}

	var result []struct {
		Key   string
		Value interface{}
	}
	t := val.Type()

	for i := 0; i < val.NumField(); i++ {
		if !t.Field(i).IsExported() {
			continue
		}

		fieldVal := val.Field(i)
		fieldType := t.Field(i)

		// Check if the field is a struct (but not a time.Time or similar)
		if fieldVal.Kind() == reflect.Struct && fieldType.Type.PkgPath() != "time" {
			// Recursively get key-value pairs for the nested struct
			nestedPairs, err := structToKVold(fieldVal.Interface())
			if err != nil {
				return nil, err
			}
			// Add the nested struct as a single key-value pair
			result = append(result, struct {
				Key   string
				Value interface{}
			}{
				Key:   fieldType.Name,
				Value: nestedPairs,
			})
		} else {
			// Non-struct field, add directly
			result = append(result, struct {
				Key   string
				Value interface{}
			}{
				Key:   fieldType.Name,
				Value: fieldVal.Interface(),
			})
		}
	}

	return result, nil
}

func StructToKV(v interface{}, omitFields ...string) ([]struct {
	Key   string
	Value interface{}
}, error) {
	// Get reflect.Value of input struct
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("input must be a struct or pointer to a struct")
	}

	// Create map of fields to omit for O(1) lookup
	skipFields := make(map[string]bool)
	for _, field := range omitFields {
		skipFields[field] = true
	}

	var result []struct {
		Key   string
		Value interface{}
	}
	t := val.Type()

	for i := 0; i < val.NumField(); i++ {
		if !t.Field(i).IsExported() {
			continue
		}

		fieldName := t.Field(i).Name
		// Skip fields in omit list
		if skipFields[fieldName] {
			continue
		}

		fieldVal := val.Field(i)
		fieldType := t.Field(i)

		// Check if the field is a struct (but not a time.Time or similar)
		if fieldVal.Kind() == reflect.Struct && fieldType.Type.PkgPath() != "time" {
			// Recursively get key-value pairs for the nested struct
			nestedPairs, err := StructToKV(fieldVal.Interface(), omitFields...)
			if err != nil {
				return nil, err
			}
			result = append(result, struct {
				Key   string
				Value interface{}
			}{
				Key:   fieldName,
				Value: nestedPairs,
			})
		} else {
			// Non-struct field, add directly
			result = append(result, struct {
				Key   string
				Value interface{}
			}{
				Key:   fieldName,
				Value: fieldVal.Interface(),
			})
		}
	}

	return result, nil
}

func StructFieldNames(v interface{}) ([]string, error) {
	// Get reflect.Type of input
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("input must be a struct or pointer to a struct")
	}

	var fieldNames []string
	for i := 0; i < t.NumField(); i++ {
		if t.Field(i).IsExported() {
			fieldNames = append(fieldNames, t.Field(i).Name)
		}
	}

	return fieldNames, nil
}
