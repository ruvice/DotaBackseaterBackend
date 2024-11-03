package vote

import (
	"context"
	"fmt"

	"github.com/ruvice/dotabackseaterbackend/model"
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
		return fmt.Errorf("Problem creating document:  %w", err)
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
		fmt.Println("FindOne failed:", err)
		if err == mongo.ErrNoDocuments {
			insertErr := r.Insert(ctx, channelID)
			if insertErr != nil {
				return insertErr
			}
		}
	}
	fmt.Println("Found document:", result)
	fmt.Println("Found votes:", result.Votes)
	return nil
}

func (r *MongoDBRepo) GetTopVote(ctx context.Context, channelID string) model.Item {
	twitchExtensionDatabase := r.Client.Database("twitchExtensionDatabase")
	channelCollection := twitchExtensionDatabase.Collection("channel")

	filter := bson.M{"channelID": channelID}
	var result VoteDocument
	err := channelCollection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		fmt.Println("FindOne failed:", err)
		return model.Item{}
	}
	fmt.Println("Found document:", result)
	fmt.Println("Found votes:", result.Votes)
	var votedItemID string
	var highestVoteCount int64
	for itemID, voteCount := range result.Votes {
		if voteCount > highestVoteCount {
			highestVoteCount = voteCount
			votedItemID = itemID
		}
	}
	var votedItem = model.Item{
		ItemID: votedItemID,
	}
	return votedItem
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

func (r *MongoDBRepo) UpdateVote(ctx context.Context, channelID string, itemID string) {
	fmt.Println("Updating Votes in MongoDB")
	twitchExtensionDatabase := r.Client.Database("twitchExtensionDatabase")
	channelCollection := twitchExtensionDatabase.Collection("channel")
	err := r.FindDocument(ctx, channelID)
	if err != nil {
		fmt.Println("Document does not exist, failed to create: ", err)
		return
	}
	var updateField = "votes." + itemID
	filter := bson.M{"channelID": channelID}
	update := bson.M{"$inc": bson.M{updateField: 1}}

	result, err := channelCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("Matched %d document(s) and modified %d document(s)\n", result.MatchedCount, result.ModifiedCount)
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
