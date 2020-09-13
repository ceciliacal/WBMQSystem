package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

// TODO: Need a singleton for this session. Every call that works with Dynamo init a new client!

func initDBClient() *dynamodb.DynamoDB {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	return dynamodb.New(sess)
}

func AddDBBot(bot Bot) {
	client := initDBClient()
	av, err := dynamodbattribute.MarshalMap(bot)
	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String("bots"),
	}
	_, err = client.PutItem(input)
	if err != nil {
		fmt.Println(err.Error())
	}
}

func removeSensor(id string) (bool, error) {

	client := initDBClient()

	//client := initDBClient()
	//av, err := dynamodbattribute.MarshalMap(sensor)
	input := &dynamodb.DeleteItemInput{
		//Item:      av,
		TableName: aws.String("sensors"),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: &id,
			},
		},
	}
	var err error
	_, err = client.DeleteItem(input)
	if err != nil {
		fmt.Println(err.Error())
		return false, nil
	}

	fmt.Println("---- Sensor "+id+" was successfully removed ")
	return true, nil
}

func removeBot(id string) (bool, error) {

	client := initDBClient()

	input := &dynamodb.DeleteItemInput{
		//Item:      av,
		TableName: aws.String("bots"),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: &id,
			},
		},
	}
	var err error
	_, err = client.DeleteItem(input)
	if err != nil {
		fmt.Println(err.Error())
		return false, nil
	}

	fmt.Println("---- Bot "+id+" was successfully removed ")
	return true, nil
}


// remove topic from bot
func removeTopic(id string) (bool, error) {

	client := initDBClient()
	expr := expression.Remove(
		expression.Name("topic"),
	)

	update, err := expression.NewBuilder().
		WithUpdate(expr).
		Build()
	if err != nil {
		return false, fmt.Errorf("failed to build update expression: %v")
	}

	input := &dynamodb.UpdateItemInput{
		TableName: aws.String("bots"),
		ExpressionAttributeNames:  update.Names(),
		ExpressionAttributeValues: update.Values(),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: &id,
			},
		},
		UpdateExpression:       update.Update(),
	}
	fmt.Println(input.String())
	_, err = client.UpdateItem(input)

	if err != nil {
		fmt.Println(err.Error())
		return false, nil
	}

	fmt.Println("Successfully unsubscribed "+id)
	return true, nil
}

func GetDBBot(id string) (Bot, error) {
	var myBot Bot

	client := initDBClient()
	params := &dynamodb.ScanInput{
		TableName: aws.String("bots"),
	}
	result, err := client.Scan(params)
	if err != nil {
		return myBot, err
	}

	for _, i := range result.Items {
		bot := Bot{}
		err = dynamodbattribute.UnmarshalMap(i, &bot)

		if bot.Id == id {
			myBot=bot
		}

		if err != nil {
			return myBot, nil
		}

	}
	return myBot, nil
}


func GetDBBots() ([]Bot, error) {
	client := initDBClient()
	params := &dynamodb.ScanInput{
		TableName: aws.String("bots"),
	}
	result, err := client.Scan(params)
	if err != nil {
		return nil, err
	}

	var botslist = []Bot{}
	for _, i := range result.Items {
		bot := Bot{}
		err = dynamodbattribute.UnmarshalMap(i, &bot)
		if err != nil {
			return nil, err
		}
		botslist = append(botslist, bot)
	}
	return botslist, nil
}

func AddDBSensor(sensor Sensor) {
	client := initDBClient()
	av, err := dynamodbattribute.MarshalMap(sensor)
	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String("sensors"),
	}
	_, err = client.PutItem(input)
	if err != nil {
		fmt.Println(err.Error())
	}
}

func GetDBSensors() ([]Sensor, error) {
	client := initDBClient()
	params := &dynamodb.ScanInput{
		TableName: aws.String("sensors"),
	}
	result, err := client.Scan(params)
	if err != nil {
		return nil, err
	}

	var sensorslist = []Sensor{}
	for _, i := range result.Items {
		sensor := Sensor{}
		err = dynamodbattribute.UnmarshalMap(i, &sensor)
		if err != nil {
			return nil, err
		}
		sensorslist = append(sensorslist, sensor)
	}
	return sensorslist, nil
}