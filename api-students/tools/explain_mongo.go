package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"go.mongodb.org/mongo-driver/v2/bson"

	"api-students/config"
	"api-students/database"
)

func main() {
	config.LoadEnv()
	client, err := database.NewMongoClient(context.Background())
	if err != nil {
		fmt.Printf("Error mongo: %v\n", err)
		os.Exit(1)
	}
	defer client.Disconnect(context.Background())

	db := client.Database(config.GetEnv("MONGO_DATABASE", "praktikum_backend"))

	// Explain on student_portfolios unique index (student_id: 1)
	cmd := bson.D{
		{Key: "explain", Value: bson.D{
			{Key: "find", Value: "student_portfolios"},
			{Key: "filter", Value: bson.D{{Key: "student_id", Value: 2}}},
		}},
		{Key: "verbosity", Value: "executionStats"},
	}
	raw, err := db.RunCommand(context.Background(), cmd).Raw()
	if err != nil {
		fmt.Printf("Explain error: %v\n", err)
		return
	}

	var jsonMap map[string]any
	rawBytes, _ := bson.MarshalExtJSON(raw, true, false)
	_ = json.Unmarshal(rawBytes, &jsonMap)

	execStats := jsonMap["executionStats"].(map[string]any)
	queryPlanner := jsonMap["queryPlanner"].(map[string]any)
	winningPlan := queryPlanner["winningPlan"].(map[string]any)

	fmt.Println("=== EXPLAIN EXECUTION STATS (student_portfolios) ===")
	fmt.Printf("Stage                 : %v\n", winningPlan["stage"])
	if inputStage, ok := winningPlan["inputStage"].(map[string]any); ok {
		fmt.Printf("Input Stage           : %v\n", inputStage["stage"])
		fmt.Printf("Index Name            : %v\n", inputStage["indexName"])
	}
	fmt.Printf("nReturned             : %v\n", execStats["nReturned"])
	fmt.Printf("totalDocsExamined     : %v\n", execStats["totalDocsExamined"])
	fmt.Printf("totalKeysExamined     : %v\n", execStats["totalKeysExamined"])
	fmt.Printf("executionTimeMillis   : %v ms\n", execStats["executionTimeMillis"])

	// Check sample document in student_portfolios
	var sampleRaw bson.Raw
	err = db.Collection("student_portfolios").FindOne(context.Background(), bson.D{}).Decode(&sampleRaw)
	if err == nil {
		sampleJSON, _ := bson.MarshalExtJSONIndent(sampleRaw, false, false, "", "  ")
		fmt.Println("\n=== CONTOH DOKUMEN PORTOFOLIO DI MONGODB ===")
		fmt.Println(string(sampleJSON))
	}
}
