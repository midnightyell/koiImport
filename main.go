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
type args struct {
	deleteFlag bool
	itemsDir   string
	collection string
}

func main() {
	// Define and parse command-line flags
	args := args{}
	flag.BoolVar(&args.deleteFlag, "delete", false, "Delete all data from the server")
	//flag.StringVar(&args.itemsDir, "itemsdir", "../dyn/items", "Directory to read items from")
	flag.StringVar(&args.itemsDir, "itemsdir", "", "Directory to read items from")
	flag.StringVar(&args.collection, "collection", "maps", "Collection to use for items")

	flag.Parse()

	if args.collection != "" {
		args.collection = cases.Title(language.English, cases.Compact).String(args.collection)
	}

	// Initialize context
	ctx := context.Background()

	// Create a new client
	client := koi.NewHTTPClient(target, 30*time.Second)
	_, err := client.CheckLogin(ctx, user, pass)
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
	collection, err := findOrCreateCollection(ctx, client, args.collection)
	if err != nil {
		fmt.Printf("Failed to find or create '%s' collection: %v\n", args.collection, err)
		return
	}
	fmt.Printf("Using collection: %s (ID: %s)\n", collection.Title, collection.ID)

	// If itemsDir is provided, process items
	if args.itemsDir != "" {
		items, err := processJSONFiles(args.itemsDir)
		if err != nil {
			fmt.Printf("Failed to process items in directory %s: %v\n", args.itemsDir, err)
			return
		}
		fmt.Printf("Items processed successfully from directory: %s\n", args.itemsDir)
		fmt.Printf("Total items decoded: %d\n", len(items))
		IRI := collection.IRI() // Set the collection IRI for each item
		for i, item := range items {
			fmt.Printf("Item %d: ID=%s, Name=%s\n", i+1, item.ID, item.Name)
			item.Collection = &IRI // Set the collection ID for each item
			_, err = item.Create(ctx, client)
			if err != nil {
				fmt.Printf("Failed to create item %s: %v\n", item.Name, err)
				client.PrintError(ctx) // Print error details
				continue
			}
		}
	} else {
		fmt.Println("No items directory provided, skipping item processing")
	}

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
			if strings.ToLower(collection.Title) == collectionName {
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
