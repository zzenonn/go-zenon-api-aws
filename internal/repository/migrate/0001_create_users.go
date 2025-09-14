package migrate

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

const (
	TableName = "users"
	Version   = "20250405000000_users_table"
)

type CreateUsersTable struct{}

func (m *CreateUsersTable) Version() string {
	return Version
}

func (m *CreateUsersTable) TableName() string {
	return TableName
}

func (m *CreateUsersTable) Up(ctx context.Context, client *dynamodb.Client) error {
	log.Infof("Creating DynamoDB table: %s", TableName)
	
	input := &dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("pk"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("pk"),
				KeyType:       types.KeyTypeHash,
			},
		},
		TableName: aws.String(TableName),
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(5),
			WriteCapacityUnits: aws.Int64(5),
		},
	}

	_, err := client.CreateTable(ctx, input)
	if err != nil {
		log.Errorf("Failed to create table %s: %v", TableName, err)
		return err
	}

	log.Infof("Waiting for table %s to become active...", TableName)
	waiter := dynamodb.NewTableExistsWaiter(client)
	err = waiter.Wait(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(TableName),
	}, 5*time.Minute)

	if err != nil {
		log.Errorf("Table %s failed to become active: %v", TableName, err)
		return err
	}

	log.Infof("Table %s created successfully", TableName)

	// Create default admin user
	log.Info("Creating default admin user")
	err = m.createDefaultUser(ctx, client)
	if err != nil {
		log.Errorf("Failed to create default admin user: %v", err)
		return err
	}

	log.Info("Default admin user created successfully")
	return nil
}

func (m *CreateUsersTable) Down(ctx context.Context, client *dynamodb.Client) error {
	log.Infof("Deleting DynamoDB table: %s", TableName)
	
	input := &dynamodb.DeleteTableInput{
		TableName: aws.String(TableName),
	}

	_, err := client.DeleteTable(ctx, input)
	if err != nil {
		log.Errorf("Failed to delete table %s: %v", TableName, err)
		return err
	}

	log.Infof("Waiting for table %s to be completely deleted...", TableName)
	waiter := dynamodb.NewTableNotExistsWaiter(client)
	err = waiter.Wait(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(TableName),
	}, 5*time.Minute)

	if err != nil {
		log.Errorf("Table %s failed to be completely deleted: %v", TableName, err)
		return err
	}

	log.Infof("Table %s deleted successfully", TableName)
	return nil
}

func (m *CreateUsersTable) createDefaultUser(ctx context.Context, client *dynamodb.Client) error {
	// Hash the admin password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin"), 10)
	if err != nil {
		return err
	}

	// Create admin user item
	item := map[string]types.AttributeValue{
		"pk":              &types.AttributeValueMemberS{Value: "admin"},
		"hashed_password": &types.AttributeValueMemberB{Value: hashedPassword},
	}

	// Put the admin user into the table
	_, err = client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(TableName),
		Item:      item,
	})

	return err
}
