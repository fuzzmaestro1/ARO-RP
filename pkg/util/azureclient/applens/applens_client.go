package applens

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.
// AppLens Client created from CosmosDB Client
// (https://github.com/Azure/azure-sdk-for-go/blob/3f7acd20691214ef2cb1f0132f82115f1df01a8c/sdk/data/azcosmos/cosmos_client.go)

import (
	"context"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"

	"github.com/Azure/ARO-RP/pkg/util/pki"
)

// AppLens client is used to interact with the Azure AppLens service.
type Client struct {
	endpoint string
	pipeline runtime.Pipeline
}

// Endpoint used to create the client.
func (c *Client) Endpoint() string {
	return c.endpoint
}

// NewClient creates a new instance of AppLens client with Azure AD access token authentication. It uses the default pipeline configuration.
// endpoint - The AppLens service endpoint to use.
// issuerUrlTemplate - The URL template to fetch the certs used by AppLens example: https://issuer.pki.azure.com/dsms/issuercertificates?getissuersv3&caName=%s
// caName - Is the certificate authority used by AppLens example: ame
// cred - The credential used to authenticate with the AppLens service.
// options - Optional AppLens client options.  Pass nil to accept default values.
func NewClient(endpoint, issuerUrlTemplate, caName, scope string, cred azcore.TokenCredential, o *ClientOptions) (*Client, error) {
	pipeline, err := newPipeline([]policy.Policy{runtime.NewBearerTokenPolicy(cred, []string{fmt.Sprintf("%s/.default", scope)}, nil)}, o, issuerUrlTemplate, caName)

	if err != nil {
		return nil, err
	}

	return &Client{endpoint: endpoint, pipeline: *pipeline}, nil
}

func newPipeline(authPolicy []policy.Policy, options *ClientOptions, issuerUrlTemplate, caName string) (*runtime.Pipeline, error) {
	var cp *x509.CertPool = nil
	var err error = nil
	if options == nil {
		// if provided pki info fetch the correct cert pool
		// otherwise use the default of nil
		if issuerUrlTemplate != "" && caName != "" {
			cp, err = pki.GetTlsCertPool(issuerUrlTemplate, caName)
			if err != nil {
				return nil, err
			}
		}
		options = NewClientOptions(cp)
	}

	runtimePipeline := runtime.NewPipeline(
		"applens", serviceLibVersion,
		runtime.PipelineOptions{
			PerCall:  []policy.Policy{},
			PerRetry: authPolicy,
		},
		&options.ClientOptions,
	)

	return &runtimePipeline, nil
}

// ListDetectors obtains the list of detectors for a service from AppLens.
// ctx - The context for the request.
// o - Options for Read operation.
func (c *Client) ListDetectors(
	ctx context.Context,
	o *ListDetectorsOptions) ([]byte, error) {
	if o == nil {
		o = &ListDetectorsOptions{}
	}

	azResponse, err := c.sendPostRequest(
		ctx,
		o,
		nil)
	if err != nil {
		return nil, err
	}

	defer azResponse.Body.Close()

	return io.ReadAll(azResponse.Body)
}

// GetDetector obtains detector information from AppLens.
// ctx - The context for the request.
// o - Options for Read operation.
func (c *Client) GetDetector(
	ctx context.Context,
	o *GetDetectorOptions) ([]byte, error) {
	if o == nil {
		o = &GetDetectorOptions{}
	}

	azResponse, err := c.sendPostRequest(
		ctx,
		o,
		nil)
	if err != nil {
		return nil, err
	}

	defer azResponse.Body.Close()

	return io.ReadAll(azResponse.Body)
}

func (c *Client) sendPostRequest(
	ctx context.Context,
	requestOptions appLensRequestOptions,
	requestEnricher func(*policy.Request)) (*http.Response, error) {
	req, err := c.createRequest(ctx, http.MethodPost, requestOptions, requestEnricher)
	if err != nil {
		return nil, err
	}

	return c.executeAndEnsureSuccessResponse(req)
}

func (c *Client) createRequest(
	ctx context.Context,
	method string,
	requestOptions appLensRequestOptions,
	requestEnricher func(*policy.Request)) (*policy.Request, error) {
	if requestOptions != nil {
		header := requestOptions.toHeader()
		ctx = runtime.WithHTTPHeader(ctx, header)
	}

	req, err := runtime.NewRequest(ctx, method, c.endpoint)
	if err != nil {
		return nil, err
	}

	if requestEnricher != nil {
		requestEnricher(req)
	}

	return req, nil
}

func (c *Client) executeAndEnsureSuccessResponse(request *policy.Request) (*http.Response, error) {
	response, err := c.pipeline.Do(request)
	if err != nil {
		return nil, err
	}

	successResponse := (response.StatusCode >= 200 && response.StatusCode < 300) || response.StatusCode == 304
	if successResponse {
		return response, nil
	}

	return nil, newAppLensError(response)
}
