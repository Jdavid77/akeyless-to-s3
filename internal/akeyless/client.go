package akeyless

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Jdavid77/akeylesstos3/internal/models"
	"github.com/akeylesslabs/akeyless-go/v3"
	"github.com/rs/zerolog/log"
)

// Client wraps the Akeyless API client
type Client struct {
	apiClient *akeyless.V2ApiService
	token     string
	basePath  string
}

// NewClient creates a new Akeyless client and authenticates using API key
func NewClient(gatewayURL, accessID, accessKey, basePath string) (*Client, error) {
	// Create API client configuration
	cfg := akeyless.NewConfiguration()
	cfg.Servers = akeyless.ServerConfigurations{
		{
			URL: gatewayURL,
		},
	}

	apiClient := akeyless.NewAPIClient(cfg).V2Api

	// Authenticate with API key
	ctx := context.Background()
	authBody := akeyless.Auth{
		AccessId:   akeyless.PtrString(accessID),
		AccessKey:  akeyless.PtrString(accessKey),
		AccessType: akeyless.PtrString("api_key"),
	}

	authResult, _, err := apiClient.Auth(ctx).Body(authBody).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate with Akeyless: %w", err)
	}

	token := authResult.GetToken()
	if token == "" {
		return nil, fmt.Errorf("authentication failed: no token received")
	}

	log.Info().Msg("Successfully authenticated with Akeyless")

	return &Client{
		apiClient: apiClient,
		token:     token,
		basePath:  basePath,
	}, nil
}

// ListAllSecrets recursively lists all secrets from the base path
func (c *Client) ListAllSecrets(ctx context.Context) ([]models.SecretItem, error) {
	log.Info().Str("base_path", c.basePath).Msg("Starting to list all secrets")

	allSecrets := make([]models.SecretItem, 0)
	pathsToExplore := []string{c.basePath}

	for len(pathsToExplore) > 0 {
		currentPath := pathsToExplore[0]
		pathsToExplore = pathsToExplore[1:]

		log.Info().Str("path", currentPath).Msg("Listing items in path")

		listBody := akeyless.ListItems{
			Token: &c.token,
		}

		// Only set Path if it's not root
		if currentPath != "/" && currentPath != "" {
			listBody.Path = akeyless.PtrString(currentPath)
			log.Info().Str("filter_path", currentPath).Msg("Setting path filter")
		} else {
			log.Info().Msg("Listing all items (no path filter)")
		}

		listResult, _, err := c.apiClient.ListItems(ctx).Body(listBody).Execute()
		if err != nil {
			log.Error().Err(err).Str("path", currentPath).Msg("Failed to list items in path, skipping")
			continue
		}

		items := listResult.GetItems()
		log.Info().Int("items_count", len(items)).Str("path", currentPath).Msg("Retrieved items from path")

		for _, item := range items {
			itemName := item.GetItemName()
			itemType := item.GetItemType()

			log.Info().Str("item_name", itemName).Str("item_type", itemType).Msg("Processing item")

			// If it's a folder, add to paths to explore
			if itemType == "folder" {
				pathsToExplore = append(pathsToExplore, itemName)
				log.Info().Str("folder", itemName).Msg("Found folder, will explore")
			} else if isSecretType(itemType) {
				// If it's a secret, add to results
				allSecrets = append(allSecrets, models.SecretItem{
					ItemName: itemName,
					ItemType: itemType,
				})
				log.Info().Str("secret", itemName).Str("type", itemType).Msg("Found secret")
			} else {
				log.Warn().Str("item_name", itemName).Str("item_type", itemType).Msg("Skipping item - not a recognized secret type")
			}
		}
	}

	log.Info().Int("count", len(allSecrets)).Msg("Finished listing all secrets")
	return allSecrets, nil
}

// GetSecretValue retrieves the value of a specific secret
func (c *Client) GetSecretValue(ctx context.Context, secretPath string) (*models.Secret, error) {
	log.Debug().Str("secret_path", secretPath).Msg("Retrieving secret value")

	getBody := akeyless.GetSecretValue{
		Token: &c.token,
		Names: []string{secretPath},
	}

	getResult, _, err := c.apiClient.GetSecretValue(ctx).Body(getBody).Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to get secret value for %s: %w", secretPath, err)
	}

	// Extract the value
	valueStr, ok := getResult[secretPath]
	if !ok {
		return nil, fmt.Errorf("secret value not found in response for %s", secretPath)
	}

	// Extract name from path (last component)
	name := extractNameFromPath(secretPath)

	return &models.Secret{
		Name:      name,
		Path:      secretPath,
		Value:     valueStr,
		Timestamp: time.Now().UTC(),
	}, nil
}

// isSecretType checks if the item type is a secret type
func isSecretType(itemType string) bool {
	// Convert to lowercase for case-insensitive comparison
	itemTypeLower := strings.ToLower(itemType)

	secretTypes := []string{
		"static-secret",
		"static_secret",
		"dynamic-secret",
		"dynamic_secret",
		"rotated-secret",
		"rotated_secret",
	}

	for _, t := range secretTypes {
		if itemTypeLower == t {
			return true
		}
	}
	return false
}

// extractNameFromPath extracts the secret name from its full path
func extractNameFromPath(path string) string {
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	if len(parts) == 0 {
		return path
	}
	return parts[len(parts)-1]
}
