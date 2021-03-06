package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"os"
	"types"
)

type AmazonWebServices struct {
	Config   *aws.Config
	Session  *session.Session
	DynamoDB dynamodbiface.DynamoDBAPI
}

// Prepare a new AWS & DynamoDB session, then configure it.
var TestAws *AmazonWebServices

func init() {
	region := os.Getenv("AWS_REGION")
	var Aws *AmazonWebServices = new(AmazonWebServices)
	Aws.Config = &aws.Config{Region: aws.String(region)}
	var err error
	Aws.Session, err = session.NewSession(Aws.Config)
	if err != nil {
		// Logs error on Amazon CloudWatch. It's sysadmin's duty to handle it.
		fmt.Println(fmt.Sprintf("Failed to connect to AWS: %s", err.Error()))
	} else {
		var svc *dynamodb.DynamoDB = dynamodb.New(Aws.Session)
		Aws.DynamoDB = dynamodbiface.DynamoDBAPI(svc)
	}
	// Instantiate a global session in TestAws
	TestAws = Aws
}

// Preparing DynamoDB Session and Calling DB's PutItem function inside.
func (self *AmazonWebServices) Put(item map[string]*dynamodb.AttributeValue) (*dynamodb.PutItemOutput, error) {
	// Get table name from OS's environment
	tableName := aws.String(os.Getenv("DEVICES_TABLE_NAME"))
	var input = &dynamodb.PutItemInput{
		Item:      item,
		TableName: tableName,
	}
	// Calling either PutItem function of interface, defined in addDevice_test.go file, or api with the input we've provided.
	// In mock case, the PutItem function of getDeviceById_test.go will be called(interface.go)
	// In real deployment environment, the PutItem function of aws (api.go) will be called.
	result, err := self.DynamoDB.PutItem(input)
	return result, err
}

// The handler function which will be first started from main function.
func AddDevice(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// First & foremost we have to validate user input.
	NewDevice, err := ValidateInputs(request)
	// if inputs are not suitable, return HTTP error code 400.
	if err != nil {
		return events.APIGatewayProxyResponse{
			Body:       "" + err.Error(),
			StatusCode: 400,
		}, nil
	}

	// Serialization/Encoding "NewDevice" in "item" for using in DynamoDB functions.
	item, _ := dynamodbattribute.MarshalMap(NewDevice)

	// Till now the user have provided a valid data input.
	// Let's add it to the DynamoDB table.
	_, err = TestAws.Put(item)

	// If internal database errors occurred, return HTTP error code 500.
	if err != nil {
		return events.APIGatewayProxyResponse{
			Body:       "Internal Server Error\nDatabase error.",
			StatusCode: 500,
		}, nil
	}

	// Serialization/Encoding "NewDevice" to JSON.
	jsonResponse, _ := json.Marshal(NewDevice)
	return events.APIGatewayProxyResponse{
		Body: string(jsonResponse),
		// Everything looks fine, return HTTP 201
		StatusCode: 201,
	}, nil
} // End of AddDevice function

func ValidateInputs(request events.APIGatewayProxyRequest) (types.Device, error) {
	NewDevice := types.Device{}
	ErrorMessage := ""

	if len(request.Body) == 0 {
		ErrorMessage = "No inputs provided, please provide inputs in JSON format."
		return types.Device{}, errors.New(ErrorMessage)
	}

	// De-serialize "request.Body" which is in JSON format into "NewDevice" in Go object.
	var err = json.Unmarshal([]byte(request.Body), &NewDevice)

	if err != nil {
		ErrorMessage = "Wrong format: Inputs must be a valid JSON."
		return types.Device{}, errors.New(ErrorMessage)
	}

	if len(NewDevice.ID) == 0 {
		ErrorMessage = "Missing field: ID"
		return types.Device{}, errors.New(ErrorMessage)
	}

	if len(NewDevice.DeviceModel) == 0 {
		ErrorMessage = "Missing field: Device Model"
		return types.Device{}, errors.New(ErrorMessage)
	}

	if len(NewDevice.Name) == 0 {
		ErrorMessage = "Missing field: Name"
		return types.Device{}, errors.New(ErrorMessage)
	}

	if len(NewDevice.Note) == 0 {
		ErrorMessage = "Missing field: Note"
		return types.Device{}, errors.New(ErrorMessage)
	}

	if len(NewDevice.Serial) == 0 {
		ErrorMessage = "Missing field: Serial"
		return types.Device{}, errors.New(ErrorMessage)
	}

	// Everything looks fine, return created NewDevice in Go struct.
	return NewDevice, nil
} // End of ValidateInputs function.

func main() {
	lambda.Start(AddDevice)
}
