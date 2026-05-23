package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	ctypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

type CognitoClient struct {
	client       *cognitoidentityprovider.Client
	clientID     string
	clientSecret string
}

func NewCognitoClient(client *cognitoidentityprovider.Client, clientID, clientSecret string) *CognitoClient {
	return &CognitoClient{
		client:       client,
		clientID:     clientID,
		clientSecret: clientSecret,
	}
}

func (c *CognitoClient) SignUp(ctx context.Context, username, password, email string) error {
	hash := c.secretHash(username)
	_, err := c.client.SignUp(ctx, &cognitoidentityprovider.SignUpInput{
		ClientId:   aws.String(c.clientID),
		Username:   aws.String(username),
		Password:   aws.String(password),
		SecretHash: aws.String(hash),
		UserAttributes: []ctypes.AttributeType{
			{Name: aws.String("email"), Value: aws.String(email)},
		},
	})
	return err
}

func (c *CognitoClient) ConfirmSignUp(ctx context.Context, username, code string) error {
	hash := c.secretHash(username)
	_, err := c.client.ConfirmSignUp(ctx, &cognitoidentityprovider.ConfirmSignUpInput{
		ClientId:         aws.String(c.clientID),
		Username:         aws.String(username),
		ConfirmationCode: aws.String(code),
		SecretHash:       aws.String(hash),
	})
	return err
}

type AuthResult struct {
	AccessToken  string
	IDToken      string
	RefreshToken string
}

func (c *CognitoClient) SignIn(ctx context.Context, username, password string) (*AuthResult, *ctypes.ChallengeNameType, error) {
	hash := c.secretHash(username)
	resp, err := c.client.InitiateAuth(ctx, &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow: ctypes.AuthFlowTypeUserPasswordAuth,
		ClientId: aws.String(c.clientID),
		AuthParameters: map[string]string{
			"USERNAME":    username,
			"PASSWORD":    password,
			"SECRET_HASH": hash,
		},
	})
	if err != nil {
		return nil, nil, err
	}

	if resp.ChallengeName != "" {
		return nil, &resp.ChallengeName, nil
	}

	result := resp.AuthenticationResult
	if result == nil {
		return nil, nil, fmt.Errorf("no authentication result")
	}

	return &AuthResult{
		AccessToken:  aws.ToString(result.AccessToken),
		IDToken:      aws.ToString(result.IdToken),
		RefreshToken: aws.ToString(result.RefreshToken),
	}, nil, nil
}

func (c *CognitoClient) secretHash(username string) string {
	mac := hmac.New(sha256.New, []byte(c.clientSecret))
	mac.Write([]byte(username + c.clientID))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
