package wastecontainer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"golang.org/x/oauth2/clientcredentials"
)

type WasteContainer struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Location    Location `json:"location"`
	Tenant      string   `json:"tenant"`
}

type Location struct {
	Lat float64 `json:"latitude"`
	Lon float64 `json:"longitue"`
}

type FeatureCollection struct {
	Type     string    `json:"type"`
	Features []Feature `json:"features"`
}

type Feature struct {
	Type       string     `json:"type"`
	ID         string     `json:"id"`
	Geometry   Geometry   `json:"geometry"`
	Properties Properties `json:"properties"`
}

type Geometry struct {
	Type        string `json:"type"`
	Coordinates []float64
}

type Properties struct {
	Type       string    `json:"typ"`
	ID         int       `json:"objektid"`
	Department string    `json:"avdelning"`
	ModifiedAt time.Time `json:"date_modif"`
	Capacity   int32     `json:"kapacitet"`
}

var tracer = otel.Tracer("sdl-wastecontainers")

type Client struct {
	clientcredentialsConfig *clientcredentials.Config
	httpClient              http.Client
	tenant                  string
}

func New(ctx context.Context, oauthClientID, oauthClientSecret, oauthTokenURL string) (*Client, error) {
	oauthConfig := &clientcredentials.Config{
		ClientID:     oauthClientID,
		ClientSecret: oauthClientSecret,
		TokenURL:     oauthTokenURL,
	}

	token, err := oauthConfig.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get client credentials from %s: %w", oauthConfig.TokenURL, err)
	}

	if !token.Valid() {
		return nil, fmt.Errorf("an invalid token was returned from %s", oauthTokenURL)
	}

	tenant := env.GetVariableOrDefault(ctx, "DEFAULT_TENANT", "default")

	return &Client{
		clientcredentialsConfig: oauthConfig,
		httpClient: http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
		tenant: tenant,
	}, nil
}

func (c *Client) Run(ctx context.Context, wcUrl, diwiseUrl string) error {
	log := logging.GetFromContext(ctx)

	fc, err := c.getFeatureCollection(ctx, wcUrl)
	if err != nil {
		return err
	}

	wc := c.convert(*fc)

	for _, w := range wc {
		err := c.createWasteContainer(ctx, diwiseUrl, w)
		if err != nil && errors.Is(err, ErrStatusConflict) {
			err = c.updateWasteContainer(ctx, diwiseUrl, w)
			if err != nil {
				log.Error("could not update waste container", "err", err.Error())
				continue
			}
		}
		if err != nil {
			log.Error("could not create waste container", "err", err.Error())
			continue
		}
	}

	return nil
}

func (c *Client) convert(fc FeatureCollection) []WasteContainer {
	wcs := make([]WasteContainer, 0)

	for _, f := range fc.Features {
		w := WasteContainer{
			ID:          strconv.Itoa(f.Properties.ID),
			Type:        "WasteContainer",
			Description: f.Properties.Type,
			Location: Location{
				Lat: f.Geometry.Coordinates[1],
				Lon: f.Geometry.Coordinates[0],
			},
			Tenant: c.tenant,
		}
		wcs = append(wcs, w)
	}

	return wcs
}

var ErrStatusConflict error = fmt.Errorf("conflict")
var ErrUnexpectedResponse error = fmt.Errorf("unexpected")

func (c *Client) createWasteContainer(ctx context.Context, url string, wc WasteContainer) error {
	var err error

	logger := logging.GetFromContext(ctx)

	ctx, span := tracer.Start(ctx, "create-wastecontainer")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()
	_, ctx, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

	b, err := json.Marshal(wc)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}

	token, err := c.clientcredentialsConfig.Token(ctx)
	if err != nil {
		err = fmt.Errorf("failed to get client credentials from %s: %w", c.clientcredentialsConfig.TokenURL, err)
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("%s %s", token.TokenType, token.AccessToken))

	res, err := c.httpClient.Do(req)
	if err != nil {
		log.Error("failed to post waste container to diwise", "err", err.Error(), slog.Int("statusCode", res.StatusCode))
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusConflict {
		return ErrStatusConflict
	}

	if !(res.StatusCode == http.StatusOK || res.StatusCode == http.StatusCreated) {
		log.Error("unexpected response for post waste container", slog.Int("statusCode", res.StatusCode))
		return ErrUnexpectedResponse
	}

	return nil
}

func (c *Client) updateWasteContainer(ctx context.Context, url string, wc WasteContainer) error {
	var err error

	logger := logging.GetFromContext(ctx)

	ctx, span := tracer.Start(ctx, "update-wastecontainer")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()
	_, ctx, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

	b, err := json.Marshal(wc)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, fmt.Sprintf("%s/%s", url, wc.ID), bytes.NewReader(b))
	if err != nil {
		return err
	}

	token, err := c.clientcredentialsConfig.Token(ctx)
	if err != nil {
		err = fmt.Errorf("failed to get client credentials from %s: %w", c.clientcredentialsConfig.TokenURL, err)
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("%s %s", token.TokenType, token.AccessToken))

	res, err := c.httpClient.Do(req)
	if err != nil {
		log.Error("failed to put waste container to diwise", "err", err.Error(), slog.Int("statusCode", res.StatusCode))
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Error("unexpected response for post waste container", slog.Int("statusCode", res.StatusCode))
		return ErrUnexpectedResponse
	}

	return nil
}

func (c *Client) getFeatureCollection(ctx context.Context, url string) (*FeatureCollection, error) {
	var err error

	logger := logging.GetFromContext(ctx)

	ctx, span := tracer.Start(ctx, "get-feature-collection")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()
	_, ctx, log := o11y.AddTraceIDToLoggerAndStoreInContext(span, logger, ctx)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		log.Error("failed to retrieve waste container information", "err", err.Error())
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Error("unexpected responser from server", slog.Int("statusCode", res.StatusCode))
		return nil, fmt.Errorf("unexpected response from server")
	}

	b, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error("unable to read body", "err", err.Error())
		return nil, fmt.Errorf("unable to read body")
	}

	var fc FeatureCollection
	err = json.Unmarshal(b, &fc)
	if err != nil {
		log.Error("unable to unmarshal feature collection", "err", err.Error())
		return nil, fmt.Errorf("unable to unmarshal feature collection")
	}

	return &fc, nil
}
