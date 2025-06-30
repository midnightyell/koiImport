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

// args holds command-line arguments
type cliargs struct {
	deleteFlag bool
	itemsDir   string
	collection string
	user       string
	pass       string
	target     string
}

var args cliargs

func main() {
	// Define and parse command-line flags
	//args := args{}
	flag.BoolVar(&args.deleteFlag, "delete", false, "Delete all data from the server")
	//flag.StringVar(&args.itemsDir, "itemsdir", "../dyn/items", "Directory to read items from")
	flag.StringVar(&args.itemsDir, "itemsdir", "", "Directory to read items from")
	flag.StringVar(&args.collection, "collection", "maps", "Collection to use for items")
	flag.StringVar(&args.user, "user", user, "Username for authentication")
	flag.StringVar(&args.pass, "pass", pass, "Password for authentication")
	flag.StringVar(&args.target, "target", target, "Target URL of the Koi server")
	flag.Parse()

	if args.collection != "" {
		args.collection = cases.Title(language.English, cases.Compact).String(args.collection)
	}

	ctx := context.Background()
	client := koi.NewHTTPClient(args.target, 30*time.Second)
	_, err := client.CheckLogin(ctx, args.user, args.pass)
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
			err := processItem(ctx, client, item)
			if err != nil {
				fmt.Printf("Error processing item %s: %v\n", item.Name, err)
				continue
			}
		}
	} else {
		fmt.Println("No items directory provided, skipping item processing")
	}

}

func processItem(ctx context.Context, client koi.Client, item *Item) error {

	fmt.Printf("Processing item: ID=%s, Name=%s\n", item.ID, item.Name)
	collection, err := findOrCreateCollection(ctx, client, args.collection)
	if item.Collection == nil {
		iri := collection.IRI()
		item.Collection = &iri
	}
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
	for _, feature := range item.Features {
		for _, key := range feature {
			value := string(feature[key])
			if string(key) != "" && value != "" {
				if len(value) > 50 {
					_, err = item.AddDatum(ctx, client, koi.DatumTypeTextarea, string(key), value)
				} else {
					_, err = item.AddDatum(ctx, client, koi.DatumTypeText, string(key), value)
				}
				if err != nil {
					fmt.Printf("Failed to add feature datum for item %s feature %s: %v\n", item.Name, string(key), err)
					client.PrintError(ctx) // Print error details
					return err
				}
			}
		}
	}
	for idx, pic := range item.PictureData {
		// If item has picture data, add each as a datum
		if pic.Filename != "" {
			_, err = item.AddDatum(ctx, client, koi.DatumTypeImage, fmt.Sprintf("Picture %d", idx+1), pic.Filename)

			if err != nil {
				fmt.Printf("Failed to add picture datum for item %s: %v\n", item.Name, err)
				client.PrintError(ctx) // Print error details
				return err
			}
		}
	}
	_, err = item.UploadImageByFile(ctx, client, item.PictureData[item.PhotoIndex].Filename, item.ID)
	if err != nil {
		fmt.Printf("Failed to upload image for item %s: %v\n", item.Name, err)
		client.PrintError(ctx)
		return err
	}
	fmt.Printf("Successfully processed item %s with ID %s\n", item.Name, item.ID)
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
	fmt.Printf("Successfully added datum to item %s: %s\n", item.Name, datum.ID)
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
		if !d.IsDir() || path == dir {
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

// findOrCreateMapsCollection searches for a collection named "maps" and returns it.
// If not found, it creates a new collection named "maps" and returns it.
func findOrCreateCollection(ctx context.Context, client koi.Client, collectionName string) (*koi.Collection, error) {
	// List collections to search for "maps"
	page := 1
	for {
		collections, err := client.ListCollections(ctx, page)
		if err != nil {
			return nil, fmt.Errorf("failed to list collections on page %d: %w", page, err)
		}
		if len(collections) == 0 {
			break // No more collections to check
		}

		// Check each collection for "maps"
		for _, collection := range collections {
			if strings.ToLower(collection.Title) == strings.ToLower(collectionName) {
				return collection, nil
			}
		}
		page++
	}

	// Collection not found, create a new one
	newCollection := &koi.Collection{
		Title:      collectionName,
		Visibility: koi.VisibilityPublic,
		CreatedAt:  time.Now(),
	}

	created, err := client.CreateCollection(ctx, newCollection)
	if err != nil {
		return nil, fmt.Errorf("failed to create collection '%s': %w", collectionName, err)
	}

	return created, nil
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
