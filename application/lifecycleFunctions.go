package application

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"

	"github.com/ruvice/dotabackseaterbackend/model"
	"go.mongodb.org/mongo-driver/bson"
)

func (a *App) PerformInitTasks(ctx context.Context) {
	itemMap := a.getItemsFromMongo(ctx)
	a.writeItemMapToCache(ctx, itemMap)
	a.writeItemsToCache(ctx, itemMap)
}

func (a *App) getItemsFromMongo(ctx context.Context) model.ItemMap {
	twitchExtensionDatabase := a.mongoDB.Client.Database("itemDatabase")
	channelCollection := twitchExtensionDatabase.Collection("itemsValid")

	filter := bson.M{}
	var result bson.M
	err := channelCollection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		fmt.Println("Failed to find docucment: ", err)
		return model.ItemMap{}
	}

	fmt.Println("Found document:", result)
	itemMap := make(model.ItemMap)

	// Iterate over the bson.M map and convert keys to integers
	for key, value := range result {
		// Skip the `_id` field
		if key == "_id" {
			continue
		}
		itemKey := key
		// Assert that the value is a nested object (bson.M)
		itemData, ok := value.(bson.M)
		if !ok {
			fmt.Println("Invalid value type for key:", key)
			return model.ItemMap{}
		}

		// Extract `name` and `cost` from the nested object// Extract `name`
		name, _ := itemData["name"].(string)
		itemName, _ := itemData["itemName"].(string)
		itemID, _ := itemData["id"].(string)
		// Extract `cost`, defaulting to 0 if not present or null
		var itemCost int32
		if costValue, ok := itemData["cost"]; ok && costValue != nil {
			itemCost = costValue.(int32)
		} else {
			itemCost = 0 // Default to 0 if `cost` is absent or null
		}
		item := model.Item{
			Name:     name,
			ItemName: itemName,
			Cost:     itemCost,
			ID:       itemID,
		}

		itemMap[itemKey] = item
	}
	return itemMap
}

func (a *App) writeItemMapToCache(ctx context.Context, itemMap model.ItemMap) {
	// Step 1: Convert itemMap to a slice of Items
	// Sort the 'Items' array by the 'Name' field
	var items []model.Item
	for _, item := range itemMap {
		items = append(items, item)
	}

	// Step 2: Sort the items slice by the Name field in ItemDetail
	sort.Slice(items, func(i, j int) bool {
		// Handle empty or null Name values by treating them as empty strings
		return items[i].Name < items[j].Name
	})

	// Step 3: Marshal the sorted items slice to JSON
	fmt.Println("writing:", items)
	jsonData, err := json.Marshal(items)
	if err != nil {
		fmt.Println("Failed to marshal ItemMap: ", err)
		return
	}

	// Write the JSON string to Redis
	if err := a.redisRepo.Client.Set(ctx, "itemMapCache", jsonData, 0).Err(); err != nil {
		fmt.Println("failed to write to Redis: ", err)
		return
	}

	fmt.Println("ItemMap successfully saved to Redis")
	return
}

func (a *App) writeItemsToCache(ctx context.Context, itemMap model.ItemMap) {
	for itemID, itemDetail := range itemMap {
		data, err := json.Marshal(itemDetail)
		if err != nil {
			fmt.Println("Failed to encode ItemDetail:", err)
		}
		// Generating unique key
		key := "itemID:" + itemID

		// Using transaction to make changes atomic
		txn := a.redisRepo.Client.TxPipeline()

		res := txn.Set(ctx, key, string(data), 0)
		if err := res.Err(); err != nil {
			txn.Discard()
			fmt.Println("failed to add item: ", err)
		}

		if _, err := txn.Exec(ctx); err != nil {
			fmt.Println("failed to exec:", err)
		}
	}
}

func (a *App) Cleanup(ctx context.Context) {
	log.Println("Cleaning up resources")
	if err := a.redisRepo.Client.Close(); err != nil {
		log.Printf("Failed to close Redis: %v", err)
	}

	if err := a.mongoDB.Client.Disconnect(ctx); err != nil {
		log.Printf("Failed to disconnect MongoDB: %v", err)
	}
}
