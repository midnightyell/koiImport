package main

import (
	"fmt"

	koi "gitea.local/smalloy/koiApi"
)

func PrintItems(items []*koi.Item) {
	for i, item := range items {
		fmt.Printf("[%3d] %s\n", i+1, item.Summary())
		if i+1 == len(items) {
			suffix := ""
			if i > 1 {
				suffix = "s"
			}
			fmt.Printf("\n%d %s%s\n\n", i+1, item.Type, suffix)
		}
	}
}
