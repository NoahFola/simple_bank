package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	mockdb "github.com/NoahFola/simple_bank/db/mock"
	db "github.com/NoahFola/simple_bank/db/sqlc"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

// -------------------- helpers --------------------
func decodeAccount(t *testing.T, b *bytes.Buffer) db.Account {
	t.Helper()
	var acct db.Account
	require.NoError(t, json.NewDecoder(bytes.NewReader(b.Bytes())).Decode(&acct))
	return acct
}

func decodeAccounts(t *testing.T, b *bytes.Buffer) []db.Account {
	t.Helper()
	var accts []db.Account
	require.NoError(t, json.NewDecoder(bytes.NewReader(b.Bytes())).Decode(&accts))
	return accts
}

// -------------------- GET /accounts/:id --------------------
func TestGetAccountByID(t *testing.T) {
	account := db.Account{ID: 1, Owner: "fola", Currency: "USD", Balance: 1000}

	tests := []struct {
		name          string
		accountID     int64
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name:      "OK",
			accountID: account.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).
					Times(1).Return(account, nil)
			},
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				got := decodeAccount(t, rr.Body)
				require.Equal(t, account, got)
			},
		},
		{
			name:      "NotFound",
			accountID: account.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).
					Times(1).Return(db.Account{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, rr.Code)
			},
		},
		{
			name:      "InternalError",
			accountID: account.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).
					Times(1).Return(db.Account{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rr.Code)
			},
		},
		{
			name:      "BadRequest_InvalidID",
			accountID: -1,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rr.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			store := mockdb.NewMockStore(ctrl)
			tt.buildStubs(store)

			server := NewServer(store)
			rr := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/accounts/%d", tt.accountID), nil)
			require.NoError(t, err)

			server.router.ServeHTTP(rr, req)
			tt.checkResponse(t, rr)
		})
	}
}

// -------------------- POST /accounts --------------------
func TestCreateAccount(t *testing.T) {
	reqBody := map[string]any{
		"owner":    "fola",
		"currency": "USD", // only USD or EUR allowed
	}
	want := db.Account{ID: 1, Owner: "fola", Currency: "USD", Balance: 0}

	tests := []struct {
		name          string
		body          map[string]any
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: reqBody,
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.CreateAccountParams{Owner: "fola", Currency: "USD", Balance: 0}
				store.EXPECT().CreateAccount(gomock.Any(), gomock.Eq(arg)).
					Times(1).Return(want, nil)
			},
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code) // your handler returns 200
				got := decodeAccount(t, rr.Body)
				require.Equal(t, want, got)
			},
		},
		{
			name: "BadRequest_MissingFields",
			body: map[string]any{"owner": "fola"},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateAccount(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rr.Code)
			},
		},
		{
			name: "BadRequest_InvalidCurrency",
			body: map[string]any{"owner": "fola", "currency": "NGN"},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateAccount(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rr.Code)
			},
		},
		{
			name: "InternalError_DB",
			body: reqBody,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateAccount(gomock.Any(), gomock.Any()).
					Times(1).Return(db.Account{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rr.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			store := mockdb.NewMockStore(ctrl)
			tt.buildStubs(store)

			server := NewServer(store)
			rr := httptest.NewRecorder()

			payload, _ := json.Marshal(tt.body)
			req, err := http.NewRequest(http.MethodPost, "/accounts", bytes.NewReader(payload))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			server.router.ServeHTTP(rr, req)
			tt.checkResponse(t, rr)
		})
	}
}

// -------------------- GET /accounts/ (list) --------------------
func TestListAccounts(t *testing.T) {
	accts := []db.Account{
		{ID: 1, Owner: "fola", Currency: "USD", Balance: 1000},
		{ID: 2, Owner: "bola", Currency: "EUR", Balance: 2000},
	}

	tests := []struct {
		name          string
		pageID        string
		pageSize      string
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name:     "OK_Page1_Size5",
			pageID:   "1",
			pageSize: "5",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListAccounts(gomock.Any(), db.ListAccountsParams{Limit: 5, Offset: 0}).
					Times(1).Return(accts, nil)
			},
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				got := decodeAccounts(t, rr.Body)
				require.Equal(t, accts, got)
			},
		},
		{
			name:     "OK_Page3_Size10",
			pageID:   "3",
			pageSize: "10",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListAccounts(gomock.Any(), db.ListAccountsParams{Limit: 10, Offset: 20}).
					Times(1).Return(accts[:1], nil)
			},
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				got := decodeAccounts(t, rr.Body)
				require.Equal(t, accts[:1], got)
			},
		},
		{
			name:     "BadRequest_MissingParams",
			pageID:   "",  // required
			pageSize: "5", // present
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListAccounts(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rr.Code)
			},
		},
		{
			name:     "BadRequest_OutOfRange",
			pageID:   "0",  // min=1
			pageSize: "11", // max=10
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListAccounts(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rr.Code)
			},
		},
		{
			name:     "InternalError_DB",
			pageID:   "1",
			pageSize: "5",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListAccounts(gomock.Any(), db.ListAccountsParams{Limit: 5, Offset: 0}).
					Times(1).Return(nil, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rr.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			store := mockdb.NewMockStore(ctrl)
			tt.buildStubs(store)

			server := NewServer(store)
			rr := httptest.NewRecorder()

			// NOTE: your router registered "/accounts/" (with trailing slash)
			url := "/accounts/?page_id=" + tt.pageID + "&page_size=" + tt.pageSize
			req, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			server.router.ServeHTTP(rr, req)
			tt.checkResponse(t, rr)
		})
	}
}

// -------------------- PATCH /accounts/:id --------------------
func TestUpdateAccount(t *testing.T) {
	want := db.Account{ID: 1, Owner: "fola", Currency: "USD", Balance: 5000}

	tests := []struct {
		name          string
		id            string
		body          map[string]any
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			id:   strconv.FormatInt(want.ID, 10),
			body: map[string]any{"balance": want.Balance},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.UpdateAccountParams{ID: want.ID, Balance: want.Balance}
				store.EXPECT().UpdateAccount(gomock.Any(), gomock.Eq(arg)).
					Times(1).Return(want, nil)
			},
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				got := decodeAccount(t, rr.Body)
				require.Equal(t, want, got)
			},
		},
		{
			name: "BadRequest_InvalidID",
			id:   "abc",
			body: map[string]any{"balance": 500},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateAccount(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rr.Code)
			},
		},
		{
			name: "BadRequest_MissingBalance",
			id:   "1",
			body: map[string]any{"bal": 500},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().UpdateAccount(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rr.Code)
			},
		},
		{
			name: "ErrNoRows_Currently500",
			id:   "1",
			body: map[string]any{"balance": 500},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateAccount(gomock.Any(), db.UpdateAccountParams{ID: 1, Balance: 500}).
					Times(1).Return(db.Account{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				// Your handler returns 500 on ErrNoRows right now
				require.Equal(t, http.StatusInternalServerError, rr.Code)
			},
		},
		{
			name: "InternalError_DB",
			id:   "1",
			body: map[string]any{"balance": 500},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateAccount(gomock.Any(), db.UpdateAccountParams{ID: 1, Balance: 500}).
					Times(1).Return(db.Account{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rr.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			store := mockdb.NewMockStore(ctrl)
			tt.buildStubs(store)

			server := NewServer(store)
			rr := httptest.NewRecorder()

			payload, _ := json.Marshal(tt.body)
			req, err := http.NewRequest(http.MethodPatch, "/accounts/"+tt.id, bytes.NewReader(payload))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			server.router.ServeHTTP(rr, req)
			tt.checkResponse(t, rr)
		})
	}
}

// -------------------- DELETE /accounts/:id --------------------
func TestDeleteAccount(t *testing.T) {
	tests := []struct {
		name          string
		id            string
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name: "OK_Returns200",
			id:   "1",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().DeleteAccount(gomock.Any(), int64(1)).
					Times(1).Return(nil)
			},
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code) // handler returns 200 + JSON message
			},
		},
		{
			name: "BadRequest_InvalidID",
			id:   "x",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().DeleteAccount(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rr.Code)
			},
		},
		{
			name: "NotFound",
			id:   "2",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().DeleteAccount(gomock.Any(), int64(2)).
					Times(1).Return(sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, rr.Code)
			},
		},
		{
			name: "InternalError_DB",
			id:   "3",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().DeleteAccount(gomock.Any(), int64(3)).
					Times(1).Return(sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rr.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			store := mockdb.NewMockStore(ctrl)
			tt.buildStubs(store)

			server := NewServer(store)
			rr := httptest.NewRecorder()

			req, err := http.NewRequest(http.MethodDelete, "/accounts/"+tt.id, nil)
			require.NoError(t, err)

			server.router.ServeHTTP(rr, req)
			tt.checkResponse(t, rr)
		})
	}
}
