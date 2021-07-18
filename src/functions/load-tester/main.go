package main

import (
	"context"
	"fmt"
	"load-test/handler"
	"load-test/mid"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"go.uber.org/zap"
)

func main() {

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	h := handler.NewHandler(
		lambdaHandler,
		mid.Logger(logger),
		mid.Errors(logger),
		mid.Panics(logger),
	)

	lambda.StartHandler(h)
}

type Input struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func lambdaHandler(ctx context.Context, input Input) error {
	loadTestEndpoint := os.Getenv("LOAD_TEST_ENDPOINT")
	userPoolId := os.Getenv("USER_POOL_ID")
	userPoolClientId := os.Getenv("USER_POOL_CLIENT_ID")

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		panic(err)
	}

	sm := ssm.NewFromConfig(cfg)
	gpOut, err := sm.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           aws.String(input.ID),
		WithDecryption: true,
	})
	if err != nil {
		return err
	}

	cg := cognitoidentityprovider.NewFromConfig(cfg)
	iaOut, err := cg.AdminInitiateAuth(ctx, &cognitoidentityprovider.AdminInitiateAuthInput{
		AuthFlow:   types.AuthFlowTypeAdminUserPasswordAuth,
		ClientId:   aws.String(userPoolClientId),
		UserPoolId: aws.String(userPoolId),
		AuthParameters: map[string]string{
			"USERNAME": input.Email,
			"PASSWORD": *gpOut.Parameter.Value,
		},
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodGet, loadTestEndpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Add("authorization", fmt.Sprintf("Bearer %s", *iaOut.AuthenticationResult.AccessToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("StatusCode: %v", resp.StatusCode)
	}

	return nil
}
