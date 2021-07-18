package main

import (
	"context"
	"fmt"
	"load-test/handler"
	"load-test/mid"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmTypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
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
	ID string `json:"id"`
}

type Output struct {
	Email string `json:"email"`
	ID    string `json:"id"`
}

func lambdaHandler(ctx context.Context, input Input) (Output, error) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	userPoolId := os.Getenv("USER_POOL_ID")
	userPoolClientId := os.Getenv("USER_POOL_CLIENT_ID")

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		panic("Could not load the config")
	}

	sm := secretsmanager.NewFromConfig(cfg)
	pOut, err := sm.GetRandomPassword(ctx, &secretsmanager.GetRandomPasswordInput{
		PasswordLength: 30,
	})
	if err != nil {
		return Output{}, err
	}

	email := fmt.Sprintf("okkkkzlsfeaizmiosx+%v@zqrni.com", input.ID)

	logger.Info(
		"Putting into the parameter store",
		zap.String("id", input.ID),
		zap.String("email", email),
		zap.String("password", *pOut.RandomPassword),
	)

	pStore := ssm.NewFromConfig(cfg)
	_, err = pStore.PutParameter(ctx, &ssm.PutParameterInput{
		Name:  aws.String(input.ID),
		Value: pOut.RandomPassword,
		Type:  ssmTypes.ParameterTypeSecureString,
	})
	if err != nil {
		return Output{}, err
	}

	cg := cognitoidentityprovider.NewFromConfig(cfg)
	_, err = cg.SignUp(ctx, &cognitoidentityprovider.SignUpInput{
		ClientId: aws.String(userPoolClientId),
		Password: pOut.RandomPassword,
		Username: aws.String(email),
		UserAttributes: []types.AttributeType{
			{Name: aws.String("email"), Value: aws.String(email)},
		},
	})
	if err != nil {
		return Output{}, err
	}

	_, err = cg.AdminConfirmSignUp(ctx, &cognitoidentityprovider.AdminConfirmSignUpInput{
		UserPoolId: aws.String(userPoolId),
		Username:   aws.String(email),
	})
	if err != nil {
		return Output{}, err
	}

	_, err = cg.AdminSetUserPassword(ctx, &cognitoidentityprovider.AdminSetUserPasswordInput{
		Password:   pOut.RandomPassword,
		UserPoolId: aws.String(userPoolId),
		Username:   aws.String(email),
		Permanent:  true,
	})
	if err != nil {
		return Output{}, err
	}

	return Output{Email: email, ID: input.ID}, nil
}
