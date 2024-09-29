package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCreateClientToken(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the request method is POST
		if r.Method != "POST" {
			t.Errorf("Expected 'POST' request, got '%s'", r.Method)
		}

		// Check if the request path is correct
		if r.URL.Path != "/api/tokens" {
			t.Errorf("Expected request to '/api/tokens', got '%s'", r.URL.Path)
		}

		// Check for correct headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type header 'application/json', got '%s'", r.Header.Get("Content-Type"))
		}

		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Expected Accept header 'application/json', got '%s'", r.Header.Get("Accept"))
		}

		// Decode the request body
		var requestBody map[string]string
		json.NewDecoder(r.Body).Decode(&requestBody)

		// Check if the request body contains the correct fields
		if requestBody["identity"] != "testuser" {
			t.Errorf("Expected identity 'testuser', got '%s'", requestBody["identity"])
		}
		if requestBody["secret"] != "testpassword" {
			t.Errorf("Expected secret 'testpassword', got '%s'", requestBody["secret"])
		}

		// Prepare the response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(TokenResponse{
			Token:   "test-token",
			Expires: time.Now().Add(time.Hour),
		})
	}))
	defer server.Close()

	// Create a client using the mock server URL
	client := NewClient(server.Client(), server.URL)

	// Check for errors
	if err := CreateClientToken(client, "testuser", "testpassword"); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check if the token is correct
	if client.Token != "test-token" {
		t.Errorf("Expected token 'test-token', got '%s'", client.Token)
	}

	// Check if the expiration time is in the future
	if !client.Expires.After(time.Now()) {
		t.Errorf("Expected expiration time to be in the future")
	}

	// Check if the token was set in the client
	if client.Token != "test-token" {
		t.Errorf("Expected client token to be set to 'test-token', got '%s'", client.Token)
	}
}

func TestCreateClientTokenError(t *testing.T) {
	// Create a mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "Invalid credentials"}`))
	}))
	defer server.Close()

	// Create a client using the mock server URL
	client := NewClient(server.Client(), server.URL)

	// Call the CreateClientToken function
	err := CreateClientToken(client, "wronguser", "wrongpassword")

	// Check if an error was returned
	if err == nil {
		t.Fatal("Expected an error, but got nil")
	}

	// Check if the error message is correct
	expectedError := "[/api/tokens] unexpected status code: 401, body: {\"error\": \"Invalid credentials\"}"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}

	// Check that the token was not set in the client
	if client.Token != "" {
		t.Errorf("Expected client token to be empty, got '%s'", client.Token)
	}
}

func TestCheckConnection(t *testing.T) {
	// Test successful connection
	t.Run("Successful Connection", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				t.Errorf("Expected 'GET' request, got '%s'", r.Method)
			}
			if r.URL.Path != "/api/" {
				t.Errorf("Expected request to '/api', got '%s'", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewClient(server.Client(), server.URL)
		err := client.CheckConnection()

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	})

	// Test failed connection (server error)
	t.Run("Failed Connection (Server Error)", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := NewClient(server.Client(), server.URL)
		err := client.CheckConnection()

		if err == nil {
			t.Fatal("Expected an error, but got nil")
		}

		if err.Error() != "[/api/] Unexpected status code: 500" {
			t.Errorf("Unexpected error message: %v", err)
		}
	})

	// Test connection error (network failure)
	t.Run("Connection Error (Network Failure)", func(t *testing.T) {
		client := NewClient(http.DefaultClient, "http://nonexistentdomain.example")
		err := client.CheckConnection()

		if err == nil {
			t.Fatal("Expected an error, but got nil")
		}

		// The exact error message may vary depending on the system and network configuration,
		// so we'll just check that it contains "error checking connection"
		if err.Error()[:22] != "[/api/] Error checking" {
			t.Errorf("Unexpected error message: %v", err)
		}
	})
}
