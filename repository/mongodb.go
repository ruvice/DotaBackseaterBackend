package repository

import (
	"context"
	"fmt"

	"github.com/ruvice/dotabackseaterbackend/model"
	"github.com/ruvice/dotabackseaterbackend/utils/voteErrors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type MongoDBRepo struct {
	Client *mongo.Client
}

type VoteDocument struct {
	ChannelID string           `bson:"channelID"`
	Votes     map[string]int64 `bson:"votes"`
}

func (r *MongoDBRepo) Insert(ctx context.Context, channelID string) error {
	fmt.Println("Inserting to Mongo")
	err := r.Client.Ping(ctx, readpref.Primary())
	if err != nil {
		fmt.Println("Problem reading MongoDB, ", err)
	}

	databases, err := r.Client.ListDatabaseNames(ctx, bson.M{})
	if err != nil {
		fmt.Println("Problem reading database names, ", err)
	}
	fmt.Println(databases)

	twitchExtensionDatabase := r.Client.Database("twitchExtensionDatabase")
	channelCollection := twitchExtensionDatabase.Collection("channel")
	channelResult, err := channelCollection.InsertOne(ctx, bson.D{
		{Key: "channelID", Value: channelID},
		{Key: "votes", Value: bson.M{}},
	})
	if err != nil {
		fmt.Println("Problem creating document, ", err)
		return fmt.Errorf("problem creating document:  %w", err)
	}
	fmt.Println(channelResult.InsertedID)
	return nil
}

func (r *MongoDBRepo) FindDocument(ctx context.Context, channelID string) error {
	twitchExtensionDatabase := r.Client.Database("twitchExtensionDatabase")
	channelCollection := twitchExtensionDatabase.Collection("channel")

	filter := bson.M{"channelID": channelID}
	var result VoteDocument
	err := channelCollection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		return r.handleFindError(ctx, err, channelID)
	}
	fmt.Println("Found document:", result)
	return nil
}

func (r *MongoDBRepo) handleFindError(ctx context.Context, err error, channelID string) error {
	fmt.Println("Failed to find document in FindDocument:", err)
	if err == mongo.ErrNoDocuments {
		fmt.Println("Failed to find document in FindDocument, proceeding with Insert:", err)
		insertErr := r.Insert(ctx, channelID)
		if insertErr != nil {
			fmt.Println("Failed to find document in FindDocument, failed on Insert:", err)
			return insertErr
		}
	}
	return nil
}

func (r *MongoDBRepo) GetTopVote(ctx context.Context, channelID string) (string, *voteErrors.VoteError) {
	twitchExtensionDatabase := r.Client.Database("twitchExtensionDatabase")
	channelCollection := twitchExtensionDatabase.Collection("channel")

	filter := bson.M{"channelID": channelID}
	var result VoteDocument
	err := channelCollection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		fmt.Println("FindOne failed:", err)
		voteError := voteErrors.NewError(voteErrors.CodeVotedItemNotFound, "Failed to get Voted Item from Mongo")
		return "", voteError
	}
	fmt.Println("Found document:", result)
	var votedItemID string
	var highestVoteCount int64
	for itemID, voteCount := range result.Votes {
		if voteCount > highestVoteCount {
			highestVoteCount = voteCount
			votedItemID = itemID
		}
	}
	return votedItemID, nil
}

func FindHighestVote(m map[int]int) (int, int) {
	var maxKey int
	var maxValue int

	// Iterate over the map to find the key with the largest value
	for key, value := range m {
		if value > maxValue {
			maxValue = value
			maxKey = key
		}
	}

	return maxKey, maxValue
}

func (r *MongoDBRepo) UpdateVote(ctx context.Context, channelID string, itemID string) *voteErrors.VoteError {
	fmt.Println("Updating Votes in MongoDB")
	twitchExtensionDatabase := r.Client.Database("twitchExtensionDatabase")
	channelCollection := twitchExtensionDatabase.Collection("channel")
	err := r.FindDocument(ctx, channelID)
	if err != nil {
		return r.handleUpdateError(err)
	}
	var updateField = "votes." + itemID
	filter := bson.M{"channelID": channelID}
	update := bson.M{"$inc": bson.M{updateField: 1}}

	result, err := channelCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return r.handleUpdateError(err)
	}
	fmt.Printf("Matched %d document(s) and modified %d document(s)\n", result.MatchedCount, result.ModifiedCount)
	return nil
}

func (r *MongoDBRepo) handleUpdateError(err error) *voteErrors.VoteError {
	fmt.Println("Failed in UpdateVote: ", err)
	voteError := voteErrors.NewError(voteErrors.CodeUpdateVoteError, "Failed in UpdateVote")
	return voteError
}

func (r *MongoDBRepo) ResetVotes(ctx context.Context, channelID string) {
	twitchExtensionDatabase := r.Client.Database("twitchExtensionDatabase")
	channelCollection := twitchExtensionDatabase.Collection("channel")
	filter := bson.M{"channelID": channelID}
	update := bson.M{"$set": bson.M{"votes": bson.M{}}}

	result, err := channelCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("Matched %d document(s) and modified %d document(s)\n", result.MatchedCount, result.ModifiedCount)
}

// Handling items
type ItemDetail struct {
	Name string `bson:"name" json:"name"`
	Cost int32  `bson:"cost" json:"cost"`
}

func (r *MongoDBRepo) RefreshItems(ctx context.Context) (model.ItemMap, *voteErrors.VoteError) {
	twitchExtensionDatabase := r.Client.Database("itemDatabase")
	channelCollection := twitchExtensionDatabase.Collection("itemsValid")

	filter := bson.M{}
	var result bson.M
	err := channelCollection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		fmt.Println("Failed to find docucment: ", err)
		voteError := voteErrors.NewError(voteErrors.CodeItemRefreshError, "Error Refreshing Items")
		return model.ItemMap{}, voteError
	}

	fmt.Println("Found document:", result)
	// Create a map to store the parsed data
	itemMap := make(model.ItemMap)

	// Iterate over the bson.M map and convert keys to integers
	for key, value := range result {
		// Skip the `_id` field
		if key == "_id" {
			continue
		}
		itemKey := key
		if err != nil {
			fmt.Println("Invalid item_id key:", key)
			voteError := voteErrors.NewError(voteErrors.CodeItemRefreshError, "Error Refreshing Items")
			return model.ItemMap{}, voteError
		}
		// Assert that the value is a nested object (bson.M)
		itemData, ok := value.(bson.M)
		if !ok {
			fmt.Println("Invalid value type for key:", key)
			voteError := voteErrors.NewError(voteErrors.CodeItemRefreshError, "Error Refreshing Items")
			return model.ItemMap{}, voteError
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
	return itemMap, nil
}

func (r *MongoDBRepo) GetItems(ctx context.Context) {
	fmt.Println("Getting items from Mongo")
}
