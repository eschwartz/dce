package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/api/response"
	"github.com/Optum/dce/pkg/common"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/gorillamux"
)

var muxLambda *gorillamux.GorillaMuxAdapter

var (
	sitePathPrefix       string
	apigwDeploymentName  string
	identityPoolID       string
	userPoolProviderName string
	userPoolClientID     string
	userPoolAppWebDomain string
	userPoolID           string
	awsCurrentRegion     string
	// Config - The configuration client
	Config common.DefaultEnvConfig
)

func init() {
	initConfig()

	log.Println("Cold start; creating router for /auth")
	authRoutes := api.Routes{
		api.Route{
			Name:        "GetAuthPage",
			Method:      "GET",
			Pattern:     "/auth",
			Queries:     api.EmptyQueryString,
			HandlerFunc: GetAuthPage,
		},
		api.Route{
			Name:        "GetAuthPageAssets",
			Method:      "GET",
			Pattern:     "/auth/public/{file}",
			Queries:     api.EmptyQueryString,
			HandlerFunc: GetAuthPageAssets,
		},
	}
	r := api.NewRouter(authRoutes)
	muxLambda = gorillamux.New(r)
}

func initConfig() {
	sitePathPrefix = Config.GetEnvVar("SITE_PATH_PREFIX", "sitePathPrefix")
	apigwDeploymentName = Config.GetEnvVar("APIGW_DEPLOYMENT_NAME", "apigwDeploymentName")
	awsCurrentRegion = Config.GetEnvVar("AWS_CURRENT_REGION", "awsCurrentRegion")

	// ssmsvc := ssm.New(sess, aws.NewConfig().WithRegion("us-west-2"))
	// keyname := "/MyService/MyApp/Dev/DATABASE_URI"
	// withDecryption := false
	// param, err := ssmsvc.GetParameter(&ssm.GetParameterInput{
	// 	Name:           &keyname,
	// 	WithDecryption: &withDecryption,
	// })

	identityPoolID = Config.GetEnvVar("PS_IDENTITY_POOL_ID", "identityPoolID")
	userPoolProviderName = Config.GetEnvVar("PS_USER_POOL_PROVIDER_NAME", "userPoolProviderName")
	userPoolClientID = Config.GetEnvVar("PS_USER_POOL_CLIENT_ID", "userPoolClientID")
	userPoolAppWebDomain = Config.GetEnvVar("PS_USER_POOL_APP_WEB_DOMAIN", "userPoolAppWebDomain")
	userPoolID = Config.GetEnvVar("PS_USER_POOL_ID", "userPoolID")
}

// Handler - Handle the lambda function
func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return muxLambda.ProxyWithContext(ctx, req)
}

func main() {
	// Send Lambda requests to the router
	lambda.Start(Handler)
}

// WriteServerErrorWithResponse - Writes a server error with the specific message.
func WriteServerErrorWithResponse(w http.ResponseWriter, message string) {
	WriteAPIErrorResponse(
		w,
		http.StatusInternalServerError,
		"ServerError",
		message,
	)
}

// WriteAPIErrorResponse - Writes the error response out to the provided ResponseWriter
func WriteAPIErrorResponse(w http.ResponseWriter, responseCode int,
	errCode string, errMessage string) {
	// Create the Error Response
	errResp := response.CreateErrorResponse(errCode, errMessage)
	apiResponse, err := json.Marshal(errResp)

	// Should most likely not return an error since response.ErrorResponse
	// is structured to be json compatible
	if err != nil {
		log.Printf("Failed to Create Valid Error Response: %s", err)
		WriteAPIResponse(w, http.StatusInternalServerError, fmt.Sprintf(
			"{\"error\":\"Failed to Create Valid Error Response: %s\"", err))
	}

	// Write an error
	WriteAPIResponse(w, responseCode, string(apiResponse))
}

// WriteAPIResponse - Writes the response out to the provided ResponseWriter
func WriteAPIResponse(w http.ResponseWriter, status int, body string) {
	w.WriteHeader(status)
	w.Write([]byte(body))
}
