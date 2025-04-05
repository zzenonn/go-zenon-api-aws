package migrate

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Simplified struct with direct field access since getters don't add value in this case
type CreateUsersTable struct {
	TableName string
	Version   string
}

func (m *CreateUsersTable) GetVersion() string {
	return m.TableName
}

func (m *CreateUsersTable) GetTableName() string {
	return m.Version
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
		TableName: aws.String(m.TableName),
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
		TableName: aws.String(m.TableName),
	}

	_, err := client.DeleteTable(ctx, input)
	return err
}
