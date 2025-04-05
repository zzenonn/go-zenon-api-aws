package migrate

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type CreateUsersTable struct{}

func (m *CreateUsersTable) Version() string {
	return "20250405000000_create_users_table"
}

func (m *CreateUsersTable) Up(ctx context.Context, client *dynamodb.Client) error {
	input := &dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("id"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("id"),
				KeyType:       types.KeyTypeHash,
			},
		},
		TableName: aws.String("users"),
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(5),
			WriteCapacityUnits: aws.Int64(5),
		},
	}

	_, err := client.CreateTable(ctx, input)
	return err
}

func (m *CreateUsersTable) Down(ctx context.Context, client *dynamodb.Client) error {
	input := &dynamodb.DeleteTableInput{
		TableName: aws.String("users"),
	}

	_, err := client.DeleteTable(ctx, input)
	return err
}
