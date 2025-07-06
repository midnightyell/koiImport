package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	koi "gitea.local/smalloy/koiApi"
)

var user = "user"
var pass = "Passw0rd"
var target = "http://192.168.30.129"
var collectionDefault = "Maps"

// args holds command-line arguments
type cliargs struct {
	deleteFlag     bool
	itemsDir       string
	collectionName string
	collection     koi.Collection
	user           string
	pass           string
	target         string
}

var args cliargs

func main() {
	// Define and parse command-line flags
	//args := args{}
	flag.BoolVar(&args.deleteFlag, "delete", false, "Delete all data from the server")
	flag.StringVar(&args.itemsDir, "itemsdir", "", "Directory to read items from")
	flag.StringVar(&args.collectionName, "collection", collectionDefault, "Collection to use for items")
	flag.Parse()

	ctx := context.Background()
	client := koi.NewHTTPClient("", 30*time.Second)
	_, err := client.CheckLogin(ctx, "", "")
	if err != nil {
		fmt.Printf("Login failed: %v\n", err)
		return
	}

	// Delete all data if --delete flag is provided
	if args.deleteFlag {
		err := client.DeleteAllData(ctx)
		if err != nil {
			fmt.Printf("Failed to delete all data: %v\n", err)
			return
		}
		fmt.Println("All data deleted successfully")
		return
	}

	args.collection.Title = cases.Title(language.English, cases.Compact).String(args.collectionName)
	collectionList, err := client.ListCollections(ctx)
	for _, c := range collectionList {
		if c.Title == args.collectionName {
			args.collection = *c
		}
	}

	// todo check for more than 1 match
	// If itemsDir is provided, process items
	if args.itemsDir != "" {
		items, err := processJSONFiles(args.itemsDir)
		if err != nil {
			fmt.Printf("Failed to process items in directory %s: %v\n", args.itemsDir, err)
			return
		}
		fmt.Printf("Items processed successfully from directory: %s\n", args.itemsDir)
		fmt.Printf("Total items decoded: %d\n", len(items))
		for i, item := range items {
			fmt.Printf("Item %d: ID=%s, Name=%s\n", i+1, item.ID, item.Name)
		}

		for _, item := range items {
			if item == nil {
				fmt.Println("Skipping nil item")
				continue
			}
			err := addItemToKoi(ctx, client, item)
			if err != nil {
				fmt.Printf("Error processing item %s: %v\n", item.Name, err)
				break
			}
		}
	} else {
		fmt.Println("No items directory provided, skipping item processing")
	}

}

func addItemToKoi(ctx context.Context, client koi.Client, item *Item) error {

	fmt.Printf("Adding item to koi: EbayID=%s, Name=%s\n", item.EbayID, item.Name)
	collection := args.collection

	fmt.Printf("Using collection: %s (ID: %s)\n", collection.Title, collection.ID)

	createdItem, err := item.Create(ctx, client)
	if err != nil {
		fmt.Printf("Failed to create item %s: %v\n", item.Name, err)
		client.PrintError(ctx) // Print error details
		return err
	}
	item.Item = *createdItem // Update the local item with the created item

	// Optionally, update fields of your local item from createdItem if needed
	if item.URL != "" {
		// If item has a URL, add it as a datum
		_, err = item.AddDatum(ctx, client, koi.DatumTypeLink, "URL", item.URL)
		if err != nil {
			fmt.Printf("Failed to add URL datum for item %s: %v\n", item.Name, err)
			client.PrintError(ctx) // Print error details
			return err
		}
	}
	if item.EbayID != "" {
		// If item has an eBay ID, add it as a datum
		_, err = item.AddDatum(ctx, client, koi.DatumTypeText, "eBay ID", item.EbayID)
		if err != nil {
			fmt.Printf("Failed to add eBay ID datum for item %s: %v\n", item.Name, err)
			client.PrintError(ctx) // Print error details
			return err
		}
	}
	if item.PriceOriginal != "" {
		// If item has an original price, add it as a datum
		_, err = item.AddDatum(ctx, client, koi.DatumTypeText, "Original Price", item.PriceOriginal)
		if err != nil {
			fmt.Printf("Failed to add original price datum for item %s: %v\n", item.Name, err)
			client.PrintError(ctx) // Print error details
			return err
		}
	}
	if item.SellerName != "" {
		// If item has a seller name, add it as a datum
		_, err = item.AddDatum(ctx, client, koi.DatumTypeText, "Seller Name", item.SellerName)
		if err != nil {
			fmt.Printf("Failed to add seller name datum for item %s: %v\n", item.Name, err)
			client.PrintError(ctx) // Print error details
			return err
		}
	}
	if item.SellerURL != "" {
		// If item has a seller URL, add it as a datum
		_, err = item.AddDatum(ctx, client, koi.DatumTypeText, "Seller URL", item.SellerURL)
		if err != nil {
			fmt.Printf("Failed to add seller URL datum for item %s: %v\n", item.Name, err)
			client.PrintError(ctx) // Print error details
			return err
		}
	}
	if item.DescriptionText != "" {
		// If item has a description text, add it as a datum
		_, err = item.AddDatum(ctx, client, koi.DatumTypeTextarea, "Description Text", item.DescriptionText)
		if err != nil {
			fmt.Printf("Failed to add description text datum for item %s: %v\n", item.Name, err)
			client.PrintError(ctx) // Print error details
			return err
		}
	}
	for k, v := range item.Features {
		if k != "" && v != "" {
			if len(v) > 50 {
				_, err = item.AddDatum(ctx, client, koi.DatumTypeTextarea, k, v)
			} else {
				_, err = item.AddDatum(ctx, client, koi.DatumTypeText, k, v)
			}
			if err != nil {
				fmt.Printf("Failed to add feature datum for item %s feature %s: %v\n", item.EbayID, k, err)
				client.PrintError(ctx) // Print error details
				return err
			}
		}
	}

	for idx, pic := range item.PictureData {
		// If item has picture data, add each as a datum
		if pic.Filename != "" {
			_, err = item.AddDatum(ctx, client, koi.DatumTypeImage, fmt.Sprintf("Picture %d", idx+1), args.itemsDir+"/../"+pic.Filename)

			if err != nil {
				fmt.Printf("Failed to add picture datum for item %s: %v\n", item.EbayID, err)
				client.PrintError(ctx) // Print error details
				return err
			}
		}
	}
	_, err = item.UploadImageByFile(ctx, client, args.itemsDir+"/../"+item.PictureData[item.PhotoIndex].Filename, item.ID)
	if err != nil {
		fmt.Printf("Failed to upload image for item %s: %v\n", item.EbayID, err)
		client.PrintError(ctx)
		return err
	}
	fmt.Printf("Successfully processed item %s with ID %s\n", item.EbayID, item.ID)
	return nil
}

func (item *Item) AddDatum(ctx context.Context, client koi.Client, datumType string, Label string, Value string) (*koi.Datum, error) {
	iri := item.IRI()
	var d koi.Datum = koi.Datum{
		Item:      &iri,
		DatumType: datumType,
		Label:     Label,
	}
	datum, err := d.Create(ctx, client)
	if err != nil {
		fmt.Printf("Failed to create datum for item %s: %v\n", item.Name, err)
		client.PrintError(ctx) // Print error details
		return nil, err
	}

	switch datumType {
	case koi.DatumTypeVideo:
		datum, err = datum.UploadVideoByFile(ctx, client, "", datum.ID)
	case koi.DatumTypeFile:
		datum, err = datum.UploadFileByFile(ctx, client, Value, datum.ID)
	case koi.DatumTypeImage:
		datum, err = datum.UploadImageByFile(ctx, client, Value, datum.ID)
	case koi.DatumTypeSign:
		datum, err = datum.UploadFileByFile(ctx, client, Value, datum.ID)
	default:
		datum.Value = &Value
		datum, err = datum.Update(ctx, client)
	}
	if err != nil {
		fmt.Printf("Failed to upload datum for item %s: %v\n", item.Name, err)
		client.PrintError(ctx) // Print error details
		return nil, err
	}
	fmt.Printf("Successfully added datum to item %s: Type: %s, ID: %s, Label: %s\n", item.EbayID, datumType, datum.ID, datum.Label)
	return datum, nil
}

// processJSONFiles iterates through subdirectories in dir, decodes S/S.json into Item structs.
func processJSONFiles(dir string) ([]*Item, error) {
	var items []*Item

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a directory or is the root directory
		if !d.IsDir() || path == dir || strings.HasSuffix(path, ".git") {
			return nil
		}

		_, _err := os.Stat(fmt.Sprintf("%s/.skip", path))
		if _err == nil {
			fmt.Printf("Skipping %s because .skip file in directory\n", filepath.Base(path))
			return nil
		}

		// Construct path to S/S.json
		jsonPath := filepath.Join(path, filepath.Base(path)+".json")

		// Open the JSON file
		file, err := os.Open(jsonPath)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("JSON file not found: %s\n", jsonPath)
				return nil // Continue to next directory
			}
			return fmt.Errorf("error opening %s: %w", jsonPath, err)
		}
		defer file.Close()

		// Decode the JSON file into an Item struct
		var item Item
		decoder := json.NewDecoder(file)
		if err := decoder.Decode(&item); err != nil {
			return fmt.Errorf("error decoding %s: %w", jsonPath, err)
		}

		fmt.Printf("Successfully decoded %s\n", jsonPath)

		val, err := readIntFromFile(filepath.Join(path, ".index"))
		if err == nil {
			item.PhotoIndex = val
		}

		items = append(items, &item)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error processing directories: %w", err)
	}

	return items, nil
}

func readIntFromFile(filePath string) (int, error) {
	// Check if file exists
	_, err := os.Stat(filePath)
	if err != nil {
		return 0, err
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return 0, err
	}

	// Convert content to integer
	value, err := strconv.Atoi(strings.TrimSpace(string(content)))
	if err != nil {
		return 0, err
	}

	return value, nil
}
