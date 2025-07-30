package checker_test

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/Houeta/chrono-flow/internal/models"
	"github.com/Houeta/chrono-flow/internal/repository"
	"github.com/Houeta/chrono-flow/internal/services/checker"
	"github.com/Houeta/chrono-flow/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type errReader int

func (errReader) Read(_ []byte) (int, error) {
	return 0, errors.New("test error: forced read failure")
}

func TestChecker_CheckForUpdates(t *testing.T) {
	ctx := t.Context()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	product1Old := models.Product{Model: "A1", Price: "100"}
	product1New := models.Product{Model: "A1", Price: "110"}
	product2 := models.Product{Model: "B2", Price: "200"}
	product3 := models.Product{Model: "C3", Price: "300"}

	oldState := &models.State{
		PageHash: "d7531c3b8364299905267349982070a9b5894b9ee25b8798158a1f87912f2c83", // "hash_old"
		Products: []models.Product{product1Old, product2},
	}

	testCases := []struct {
		name            string
		setupMocks      func(mParser *mocks.HTMLParser, mRepo *mocks.StateRepository)
		expectedChanges *models.Changes
		expectError     bool
	}{
		{
			name: "Success: All types of changes found",
			setupMocks: func(mParser *mocks.HTMLParser, mRepo *mocks.StateRepository) {
				newHTML := `<html><body>new content</body></html>`
				mockHTTPResponse := &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(newHTML))),
				}
				mParser.On("GetHTMLResponse", ctx).Return(mockHTTPResponse, nil).Once()
				mRepo.On("GetState", ctx).Return(oldState, nil).Once()

				newProducts := []models.Product{product1New, product3}
				mParser.On("ParseTableResponse", ctx, mock.Anything).Return(newProducts, nil).Once()

				mRepo.On("UpdateState", ctx, mock.AnythingOfType("*models.State")).Return(nil).Once()
			},
			expectedChanges: &models.Changes{
				Added:   []models.Product{product3},
				Removed: []models.Product{product2},
				Changed: []models.ChangeInfo{{Old: product1Old, New: product1New}},
			},
			expectError: false,
		},
		{
			name: "No change: The page hash has not changed.",
			setupMocks: func(mParser *mocks.HTMLParser, mRepo *mocks.StateRepository) {
				sameHTML := `<html><body>old content</body></html>`
				mockHTTPResponse := &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(sameHTML))),
				}
				mParser.On("GetHTMLResponse", ctx).Return(mockHTTPResponse, nil).Once()

				stateWithSameHash := &models.State{
					PageHash: fmt.Sprintf("%x", sha256.Sum256([]byte(sameHTML))),
					Products: []models.Product{},
				}
				mRepo.On("GetState", ctx).Return(stateWithSameHash, nil).Once()
			},
			expectedChanges: &models.Changes{},
			expectError:     false,
		},
		{
			name: "First launch: All products added",
			setupMocks: func(mParser *mocks.HTMLParser, mRepo *mocks.StateRepository) {
				newHTML := `<html><body>new content</body></html>`
				mockHTTPResponse := &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(newHTML))),
				}
				mParser.On("GetHTMLResponse", ctx).Return(mockHTTPResponse, nil).Once()

				mRepo.On("GetState", ctx).Return(nil, repository.ErrStateNotFound).Once()

				newProducts := []models.Product{product1New, product3}
				mParser.On("ParseTableResponse", ctx, mock.Anything).Return(newProducts, nil).Once()

				expectedNewState := &models.State{
					PageHash: fmt.Sprintf("%x", sha256.Sum256([]byte(newHTML))),
					Products: newProducts,
				}
				mRepo.On("UpdateState", ctx, expectedNewState).Return(nil).Once()
			},
			expectedChanges: &models.Changes{
				Added: []models.Product{product1New, product3},
			},
			expectError: false,
		},
		{
			name: "Error: Parser cannot retrieve page",
			setupMocks: func(mParser *mocks.HTMLParser, _ *mocks.StateRepository) {
				mParser.On("GetHTMLResponse", ctx).Return(nil, errors.New("network error")).Once()
			},
			expectedChanges: nil,
			expectError:     true,
		},
		{
			name: "Error: Repository cannot update state",
			setupMocks: func(mParser *mocks.HTMLParser, mRepo *mocks.StateRepository) {
				newHTML := `<html><body>new content</body></html>`
				mockHTTPResponse := &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(newHTML))),
				}
				mParser.On("GetHTMLResponse", ctx).Return(mockHTTPResponse, nil).Once()

				mRepo.On("GetState", ctx).Return(oldState, nil).Once()

				newProducts := []models.Product{product1New, product3}
				mParser.On("ParseTableResponse", ctx, mock.Anything).Return(newProducts, nil).Once()

				mRepo.On("UpdateState", ctx, mock.Anything).Return(errors.New("db write error")).Once()
			},
			expectedChanges: nil,
			expectError:     true,
		},
		{
			name: "Error: Repository cannot get state",
			setupMocks: func(mParser *mocks.HTMLParser, mRepo *mocks.StateRepository) {
				newHTML := `<html><body>new content</body></html>`
				mockHTTPResponse := &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(newHTML)),
				}
				mParser.On("GetHTMLResponse", ctx).Return(mockHTTPResponse, nil).Once()

				mRepo.On("GetState", ctx).Return(nil, assert.AnError).Once()
			},
			expectedChanges: nil,
			expectError:     true,
		},
		{
			name: "Error: Parser cannot parse products",
			setupMocks: func(mParser *mocks.HTMLParser, mRepo *mocks.StateRepository) {
				newHTML := `<html><body>new content</body></html>`
				mockHTTPResponse := &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(newHTML))),
				}
				mParser.On("GetHTMLResponse", ctx).Return(mockHTTPResponse, nil).Once()

				mRepo.On("GetState", ctx).Return(nil, repository.ErrStateNotFound).Once()

				mParser.On("ParseTableResponse", ctx, mock.Anything).Return(nil, assert.AnError).Once()
			},
			expectedChanges: nil,
			expectError:     true,
		},
		{
			name: "Error: failed to read response body",
			setupMocks: func(mParser *mocks.HTMLParser, _ *mocks.StateRepository) {
				mockHTTPResponse := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(errReader(0))}
				mParser.On("GetHTMLResponse", ctx).Return(mockHTTPResponse, nil).Once()
			},
			expectedChanges: nil,
			expectError:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockParser := new(mocks.HTMLParser)
			mockRepo := new(mocks.StateRepository)
			tc.setupMocks(mockParser, mockRepo)

			updateChecker := checker.NewChecker(logger, mockParser, mockRepo)

			changes, err := updateChecker.CheckForUpdates(ctx)

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.ElementsMatch(t, tc.expectedChanges.Added, changes.Added)
				assert.ElementsMatch(t, tc.expectedChanges.Removed, changes.Removed)
				assert.ElementsMatch(t, tc.expectedChanges.Changed, changes.Changed)
			}

			mockParser.AssertExpectations(t)
			mockRepo.AssertExpectations(t)
		})
	}
}
