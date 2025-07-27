package parser

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/Houeta/chrono-flow/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRoundTripper — its a mock for http.RoundTripper.
type mockRoundTripper struct {
	response *http.Response
	err      error
}

func (m *mockRoundTripper) RoundTrip(_ *http.Request) (*http.Response, error) {
	return m.response, m.err
}

// =============================================================================
// Tests for parsing logic
// =============================================================================

func TestParseTableResponse(t *testing.T) {
	// Creating a "silent" logger that doesn't output anything during tests
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	p := NewParser(logger, "") // The URL is not important for this test.

	// Test HTML
	validHTML := `
	<html>
	<body>
		<table class="table-bordered">
			<tbody>
				<tr>
					<td>Model A</td><td>Type A</td><td>5</td><td>url_a</td><td>100.00</td>
				</tr>
				<tr>
					<td>Model B</td><td>Type B</td><td> > 3 </td><td>url_b</td><td> 250.50 </td>
				</tr>
				<tr>
					<td>this table has unsifficient number of cells</td><td></td>
				</tr>
			</tbody>
		</table>
	</body>
	</html>`

	// Expected result
	expectedProducts := []models.Product{
		{Model: "Model A", Type: "Type A", Quantity: "5", ImageURL: "url_a", Price: "100.00"},
		{Model: "Model B", Type: "Type B", Quantity: "> 3", ImageURL: "url_b", Price: "250.50"},
	}

	// Structure for table tests
	testCases := []struct {
		name          string
		inputHTML     string
		expected      []models.Product
		expectError   bool
		expectedError string
	}{
		{
			name:        "Successful parsing",
			inputHTML:   validHTML,
			expected:    expectedProducts,
			expectError: false,
		},
		{
			name:        "Empty HTML",
			inputHTML:   "",
			expected:    []models.Product(nil),
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Convert the string to io.ReadCloser
			reader := io.NopCloser(strings.NewReader(tc.inputHTML))

			products, err := p.parseTableResponse(t.Context(), reader)

			if tc.expectError {
				if err == nil {
					t.Errorf("An error was expected, but there was none.")
				}
				if !strings.Contains(err.Error(), tc.expectedError) {
					t.Errorf("Expected error '%s', received '%s'", tc.expectedError, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("An error was not expected, but it occurred: %v", err)
			}

			if !reflect.DeepEqual(products, tc.expected) {
				t.Errorf("The result is not as expected.\nExpected: %#v\nReceived: %#v", tc.expected, products)
			}
		})
	}
}

// =============================================================================
// Tests for network logic
// =============================================================================

func TestGetHTMLResponse(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := t.Context()

	testCases := []struct {
		name           string
		mockResponse   *http.Response
		mockError      error
		parserURL      string
		expectError    bool
		expectedErrMsg string
	}{
		{
			name: "Successful request (200 OK)",
			mockResponse: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("OK")),
			},
			mockError:   nil,
			parserURL:   "http://test.com",
			expectError: false,
		},
		{
			name: "Server Error (500)",
			mockResponse: &http.Response{
				StatusCode: http.StatusInternalServerError,
				Status:     "500 Internal Server Error",
				Body:       io.NopCloser(strings.NewReader("Error")),
			},
			mockError:      nil,
			parserURL:      "http://test.com",
			expectError:    true,
			expectedErrMsg: "status code error: [500]",
		},
		{
			name:           "Network error",
			mockResponse:   nil,
			mockError:      errors.New("connection failed"),
			parserURL:      "http://test.com",
			expectError:    true,
			expectedErrMsg: "connection failed",
		},
		{
			name:           "Invalid URL in parser",
			parserURL:      "://invalid-url",
			expectError:    true,
			expectedErrMsg: "failed to parse destination URL",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Creating a mock client with a customized response
			mockClient := &http.Client{
				Transport: &mockRoundTripper{
					response: tc.mockResponse,
					err:      tc.mockError,
				},
			}

			// Creating a parser with a mock client
			p := NewParser(logger, tc.parserURL)
			p.client = mockClient

			resp, err := p.getHTMLResponse(ctx)

			if tc.expectError {
				if err == nil {
					t.Fatalf("An error was expected, but there was none.")
				}
				if !strings.Contains(err.Error(), tc.expectedErrMsg) {
					t.Errorf("Expected error '%s', received '%s'", tc.expectedErrMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("An error was not expected, but it occurred: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, received %d", resp.StatusCode)
			}
		})
	}
}

// =============================================================================
// Integration test for the main method
// =============================================================================

func TestParseProducts(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := t.Context()

	// Preparing a successful HTML response
	successHTML := `
	<table class="table-bordered">
		<tbody>
			<tr><td>Model 1</td><td>Type 1</td><td>1</td><td>url1</td><td>99.99</td></tr>
		</tbody>
	</table>`

	// We configure a mock client to return this response
	mockClient := &http.Client{
		Transport: &mockRoundTripper{
			response: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte(successHTML))),
			},
		},
	}

	p := NewParser(logger, "http://valid-url.com")
	p.client = mockClient

	products, err := p.ParseProducts(ctx)
	if err != nil {
		t.Fatalf("ParseProducts() returned an error: %v", err)
	}

	expected := []models.Product{
		{Model: "Model 1", Type: "Type 1", Quantity: "1", ImageURL: "url1", Price: "99.99"},
	}

	if !reflect.DeepEqual(products, expected) {
		t.Errorf("The result is not as expected.\nExpected: %+v\nReceived:    %+v", expected, products)
	}
}

func TestParseProducts_ResponseError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := t.Context()

	p := NewParser(logger, ";;/invalid-url")

	products, err := p.ParseProducts(ctx)

	assert.Nil(t, products)
	require.Error(t, err)
	require.ErrorContains(t, err, "failed to get html response")
}
