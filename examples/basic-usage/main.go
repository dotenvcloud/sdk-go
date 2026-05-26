package main

import (
	"context"
	"fmt"
	"log"
	"os"

	dotenv "github.com/lostlink/dotenv-sdk-go"
)

func main() {
	// Get API key from environment
	apiKey := os.Getenv("DOTENV_API_KEY")
	if apiKey == "" {
		log.Fatal("DOTENV_API_KEY environment variable is required")
	}

	// Get organization from environment
	organization := os.Getenv("DOTENV_ORGANIZATION")
	if organization == "" {
		log.Fatal("DOTENV_ORGANIZATION environment variable is required")
	}

	// Initialize client with API key and organization
	opts := []dotenv.ClientOption{
		dotenv.WithAPIKey(apiKey),
		dotenv.WithOrganization(organization),
	}

	// For development with https://dotenv.test
	if os.Getenv("DOTENV_BASE_URL") != "" {
		opts = append(opts, dotenv.WithBaseURL(os.Getenv("DOTENV_BASE_URL")))
	}

	client := dotenv.NewClient(opts...)

	ctx := context.Background()

	// Example 1: List organizations
	fmt.Println("=== Organizations ===")
	orgs, _, err := client.Organizations.List(ctx, nil)
	if err != nil {
		log.Printf("Error listing organizations: %v", err)
	} else {
		for _, org := range orgs {
			fmt.Printf("Organization: %s (%s) - Status: %s\n", org.Name, org.Slug, org.Status)
		}
	}

	// Example 2: List projects
	if len(orgs) > 0 {
		fmt.Println("\n=== Projects ===")
		// Projects are listed for the organization set in the client
		projects, _, err := client.Projects.List(ctx, nil)
		if err != nil {
			log.Printf("Error listing projects: %v", err)
		} else {
			for _, project := range projects {
				fmt.Printf("Project: %s (%s) - Secrets: %d\n",
					project.Name, project.Slug, project.SecretCount)
			}
		}

		// Example 3: Get secrets for a project
		if len(projects) > 0 {
			fmt.Println("\n=== Secrets ===")
			secretsResp, _, err := client.Secrets.GetProjectSecrets(ctx, projects[0].Slug, "", "")
			if err != nil {
				log.Printf("Error getting secrets: %v", err)
			} else {
				fmt.Printf("Found %d levels of secrets\n", len(secretsResp.Data.Attributes.Levels))
				for level, data := range secretsResp.Data.Attributes.Levels {
					fmt.Printf("  - Level: %s (encrypted: %v, source: %s)\n",
						level, data.Encrypted, data.Source)
				}
			}

			// Example 4: Get encryption key
			fmt.Println("\n=== Encryption Key ===")
			encKey, _, err := client.Encryption.GetEncryptionKey(ctx, projects[0].Slug)
			if err != nil {
				log.Printf("Error getting encryption key: %v", err)
			} else {
				fmt.Printf("Encryption key active: %v\n", encKey.IsActive)
				fmt.Printf("Client managed: %v\n", encKey.IsClientManaged)
			}
		}
	}

	// Example 5: Error handling
	fmt.Println("\n=== Error Handling ===")
	_, _, err = client.Projects.Get(ctx, "non-existent-project")
	if err != nil {
		if dotenv.IsNotFound(err) {
			fmt.Println("Project not found (expected)")
		} else if dotenv.IsUnauthorized(err) {
			fmt.Println("Unauthorized - check your API key")
		} else if dotenv.IsRateLimited(err) {
			if rateLimitErr, ok := err.(*dotenv.ErrRateLimited); ok {
				fmt.Printf("Rate limited - retry after %d seconds\n", rateLimitErr.RetryAfter)
			}
		} else {
			fmt.Printf("Other error: %v\n", err)
		}
	}

	// Example 6: Encryption/Decryption
	fmt.Println("\n=== Encryption Example ===")

	// Generate a key
	key, err := dotenv.GenerateKey()
	if err != nil {
		log.Fatal(err)
	}

	// Encrypt some data
	secret := "my-database-password"
	encrypted, err := dotenv.Encrypt(secret, key)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Original: %s\n", secret)
	fmt.Printf("Encrypted: %s\n", encrypted)

	// Decrypt
	decrypted, err := dotenv.Decrypt(encrypted, key)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Decrypted: %s\n", decrypted)
	fmt.Printf("Match: %v\n", secret == decrypted)
}
