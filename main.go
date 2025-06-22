package main

import (
	"context"
	"fmt"
	"time"

	koi "gitea.local/smalloy/koiApi"
)

var user = "user"
var pass = "Passw0rd"
var target = "http://192.168.30.129"

func main() {
	// Initialize context
	ctx := context.Background()

	// Create a new client.
	client := koi.NewHTTPClient(target, 30*time.Second)
	_, err := client.CheckLogin(ctx, user, pass)
	if err != nil {
		fmt.Printf("Login failed: %v\n", err)
		return
	}

	err = DeleteAllTemplatesAndFields(ctx, client)
	if err != nil {
		fmt.Printf("Error deleting templates and fields: %v\n", err)
		return
	}
	fmt.Println("All templates and fields deleted successfully")

	// Create a new template
	template := &koi.Template{
		Name:      "ItemTemplate",
		CreatedAt: time.Now(),
		Type:      "Template",
	}

	// Create the template
	createdTemplate, err := template.Create(ctx, client)
	if err != nil {
		fmt.Printf("Error creating template: %v\n", err)
		return
	}
	fmt.Printf("Created template: %s\n", createdTemplate.IRI())

	// Define fields corresponding to Item struct, excluding Skip and PhotoIndex, plus Notes
	fields := []koi.Field{
		{
			Name:       "Info",
			Position:   4,
			FieldType:  koi.FieldTypeSection,
			Type:       string(koi.FieldTypeText),
			Template:   stringPtr(createdTemplate.IRI()),
			Visibility: koi.VisibilityPublic,
		},
		{
			Name:       "Title",
			Position:   4,
			FieldType:  koi.FieldTypeText,
			Type:       string(koi.FieldTypeText),
			Template:   stringPtr(createdTemplate.IRI()),
			Visibility: koi.VisibilityPublic,
		},
		{
			Name:       "Year",
			Position:   5,
			FieldType:  koi.FieldTypeText,
			Type:       string(koi.FieldTypeText),
			Template:   stringPtr(createdTemplate.IRI()),
			Visibility: koi.VisibilityPublic,
		},
		{
			Name:       "Cartographer",
			Position:   5,
			FieldType:  koi.FieldTypeText,
			Type:       string(koi.FieldTypeText),
			Template:   stringPtr(createdTemplate.IRI()),
			Visibility: koi.VisibilityPublic,
		},
		{
			Name:       "Published in",
			Position:   5,
			FieldType:  koi.FieldTypeText,
			Type:       string(koi.FieldTypeText),
			Template:   stringPtr(createdTemplate.IRI()),
			Visibility: koi.VisibilityPublic,
		},
		{
			Name:       "Features",
			Position:   8,
			FieldType:  koi.FieldTypeList,
			Type:       string(koi.FieldTypeList),
			Template:   stringPtr(createdTemplate.IRI()),
			Visibility: koi.VisibilityPublic,
		},
		{
			Name:       "DescriptionText",
			Position:   7,
			FieldType:  koi.FieldTypeTextarea,
			Type:       string(koi.FieldTypeTextarea),
			Template:   stringPtr(createdTemplate.IRI()),
			Visibility: koi.VisibilityPublic,
		},
		{
			Name:       "Notes",
			Position:   10,
			FieldType:  koi.FieldTypeTextarea,
			Type:       string(koi.FieldTypeTextarea),
			Template:   stringPtr(createdTemplate.IRI()),
			Visibility: koi.VisibilityPublic,
		},
		{
			Name:       "SellerName",
			Position:   5,
			FieldType:  koi.FieldTypeText,
			Type:       string(koi.FieldTypeText),
			Template:   stringPtr(createdTemplate.IRI()),
			Visibility: koi.VisibilityPublic,
		},
		{
			Name:       "SellerURL",
			Position:   6,
			FieldType:  koi.FieldTypeLink,
			Type:       string(koi.FieldTypeLink),
			Template:   stringPtr(createdTemplate.IRI()),
			Visibility: koi.VisibilityPublic,
		},
		{
			Name:       "EbayID",
			Position:   1,
			FieldType:  koi.FieldTypeText,
			Type:       string(koi.FieldTypeText),
			Template:   stringPtr(createdTemplate.IRI()),
			Visibility: koi.VisibilityPublic,
		},
		{
			Name:       "AuctionLink",
			Position:   1,
			FieldType:  koi.FieldTypeLink,
			Type:       string(koi.FieldTypeLink),
			Template:   stringPtr(createdTemplate.IRI()),
			Visibility: koi.VisibilityPublic,
		},
		{
			Name:       "PriceOriginal",
			Position:   2,
			FieldType:  koi.FieldTypePrice,
			Type:       string(koi.FieldTypePrice),
			Template:   stringPtr(createdTemplate.IRI()),
			Visibility: koi.VisibilityPrivate,
		},
		{
			Name:       "PriceConverted",
			Position:   3,
			FieldType:  koi.FieldTypePrice,
			Type:       string(koi.FieldTypePrice),
			Template:   stringPtr(createdTemplate.IRI()),
			Visibility: koi.VisibilityPrivate,
		},
	}

	// Create each field
	for i, field := range fields {
		createdField, err := field.Create(ctx, client)
		if err != nil {
			fmt.Printf("Error creating field %s: %v\n", field.Name, err)
			return
		}
		fmt.Printf("Created field %d: %s\n", i+1, createdField.IRI())
	}

	fmt.Println("Template and all fields created successfully")
}

// stringPtr returns a pointer to a string
func stringPtr(s string) *string {
	return &s
}

// DeleteAllTemplatesAndFields deletes all templates and their associated fields.
func DeleteAllTemplatesAndFields(ctx context.Context, client koi.Client) error {
	// Create a template instance for listing
	template := &koi.Template{}

	// List all templates
	templates, err := template.List(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to list templates: %w", err)
	}

	// Track errors but continue deleting
	var errs []error

	// Iterate through each template
	for _, tmpl := range templates {
		// Get fields associated with the template using GET /api/templates/{id}/fields
		fields, err := client.ListTemplateFields(ctx, tmpl.ID, 1)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to get fields for template %s: %w", tmpl.IRI(), err))
			continue
		}

		// Delete each field
		for _, fld := range fields {
			err := fld.Delete(ctx, client, fld.ID)
			if err != nil {
				errs = append(errs, fmt.Errorf("failed to delete field %s: %w", fld.IRI(), err))
				continue
			}
			fmt.Printf("Deleted field: %s\n", fld.IRI())
		}

		// Delete the template
		err = tmpl.Delete(ctx, client, tmpl.ID)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to delete template %s: %w", tmpl.IRI(), err))
			continue
		}
		fmt.Printf("Deleted template: %s\n", tmpl.IRI())
	}

	// If there were any errors, return them combined
	if len(errs) > 0 {
		return fmt.Errorf("encountered %d errors during deletion: %v", len(errs), errs)
	}

	fmt.Println("All templates and fields deleted successfully")
	return nil
}
