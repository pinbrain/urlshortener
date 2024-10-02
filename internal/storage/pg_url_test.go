package storage

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPgGetURL(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	urlPgStore := &URLPgStore{
		pool: mock,
	}

	type dbRes struct {
		rows []any
		err  error
	}

	type want struct {
		url string
		err error
	}

	tests := []struct {
		name  string
		urlID string
		dbRes *dbRes
		want  want
	}{
		{
			name:  "Успешно получен url",
			urlID: "shortenURL",
			dbRes: &dbRes{
				rows: []any{"some", false},
			},
			want: want{
				url: "some",
			},
		},
		{
			name:  "Ссылка не найдена",
			urlID: "shortenURL",
			dbRes: &dbRes{
				err: pgx.ErrNoRows,
			},
			want: want{
				url: "",
			},
		},
		{
			name:  "Ссылка удалена",
			urlID: "shortenURL",
			dbRes: &dbRes{
				rows: []any{"some", true},
			},
			want: want{
				url: "",
				err: ErrIsDeleted,
			},
		},
		{
			name:  "Ошибка БД",
			urlID: "shortenURL",
			dbRes: &dbRes{
				err: errors.New("db error"),
			},
			want: want{
				url: "",
				err: fmt.Errorf("failed to select url from db: %w", errors.New("db error")),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.dbRes != nil {
				mockExpectQuery := mock.ExpectQuery("SELECT .+ FROM shorten_urls WHERE .+").WithArgs(tt.urlID)
				if tt.dbRes.err != nil {
					mockExpectQuery.WillReturnError(tt.dbRes.err)
				} else if tt.dbRes.rows != nil {
					mockExpectQuery.WillReturnRows(mock.NewRows([]string{"original", "is_deleted"}).
						AddRow(tt.dbRes.rows...))
				}
			}
			url, storeErr := urlPgStore.GetURL(context.TODO(), tt.urlID)
			assert.Equal(t, tt.want.url, url)
			if tt.want.err != nil {
				assert.Equal(t, tt.want.err, storeErr)
			}
		})
	}
}

func TestPgSaveURL(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	urlPgStore := &URLPgStore{
		pool: mock,
	}

	type dbRes struct {
		rows []any
		err  error
	}

	type want struct {
		err   error
		urlID string
	}

	tests := []struct {
		name     string
		url      string
		userID   int
		dbInsert *dbRes
		dbSelect *dbRes
		want     want
	}{
		{
			name:     "Успешное сохранение ссылки",
			url:      "url_to_save",
			userID:   1,
			dbInsert: &dbRes{},
		},
		{
			name:   "Ссылка уже была сохранена",
			url:    "url_to_save",
			userID: 1,
			dbInsert: &dbRes{
				err: &pgconn.PgError{Code: pgerrcode.UniqueViolation},
			},
			dbSelect: &dbRes{
				rows: []any{"shortURL"},
			},
			want: want{
				err:   ErrConflict,
				urlID: "shortURL",
			},
		},
		{
			name:   "Ошибка БД при сохранении",
			url:    "url_to_save",
			userID: 1,
			dbInsert: &dbRes{
				err: errors.New("db error"),
			},
			want: want{
				err:   fmt.Errorf("failed to insert record to db: %w", errors.New("db error")),
				urlID: "",
			},
		},
		{
			name:   "Ошибка БД при чтении уже сохраненной ссылки",
			url:    "url_to_save",
			userID: 1,
			dbInsert: &dbRes{
				err: &pgconn.PgError{Code: pgerrcode.UniqueViolation},
			},
			dbSelect: &dbRes{
				err: errors.New("db error"),
			},
			want: want{
				err:   fmt.Errorf("failed to select existing url from db after unique conflict: %w", errors.New("db error")),
				urlID: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			insertExpectExec := mock.ExpectExec("INSERT INTO shorten_urls").
				WithArgs(tt.url, pgxmock.AnyArg(), tt.userID)
			if tt.dbInsert.err != nil {
				insertExpectExec.WillReturnError(tt.dbInsert.err)
			} else {
				insertExpectExec.WillReturnResult(pgxmock.NewResult("INSERT", 1))
			}

			if tt.dbSelect != nil {
				selectExpectQuery := mock.ExpectQuery("SELECT .+ FROM shorten_urls WHERE .+").
					WithArgs(tt.url)
				if tt.dbSelect.rows != nil {
					selectExpectQuery.WillReturnRows(mock.NewRows([]string{"shorten"}).
						AddRow(tt.dbSelect.rows...))
				} else if tt.dbSelect.err != nil {
					selectExpectQuery.WillReturnError(tt.dbSelect.err)
				}
			}

			urlID, storeErr := urlPgStore.SaveURL(context.TODO(), tt.url, tt.userID)
			if tt.want.err != nil {
				assert.Equal(t, tt.want.err, storeErr)
			} else {
				require.NoError(t, storeErr)
			}
			if tt.want.urlID != "" {
				assert.Equal(t, tt.want.urlID, urlID)
			}
		})
	}
}

func TestPgCreateUser(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	urlPgStore := &URLPgStore{
		pool: mock,
	}

	type dbRes struct {
		rows []any
		err  error
	}

	type want struct {
		err  error
		user *User
	}

	tests := []struct {
		name  string
		dbRes *dbRes
		want  want
	}{
		{
			name: "Успешное сохранение пользователя",
			dbRes: &dbRes{
				rows: []any{1},
			},
			want: want{
				user: &User{
					ID: 1,
				},
			},
		},
		{
			name: "Ошибка при сохранении пользователя",
			dbRes: &dbRes{
				err: errors.New("db error"),
			},
			want: want{
				err: fmt.Errorf("failed to create user: %w", errors.New("db error")),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExpectQuery := mock.ExpectQuery("INSERT INTO users DEFAULT VALUES RETURNING id")
			if tt.dbRes.err != nil {
				mockExpectQuery.WillReturnError(tt.dbRes.err)
			} else if tt.dbRes.rows != nil {
				mockExpectQuery.WillReturnRows(mock.NewRows([]string{"id"}).
					AddRow(tt.dbRes.rows...))
			}

			user, storeErr := urlPgStore.CreateUser(context.TODO())
			if tt.want.user != nil {
				assert.Equal(t, tt.want.user.ID, user.ID)
			}
			if tt.want.err != nil {
				assert.Equal(t, tt.want.err, storeErr)
			}
		})
	}
}

func TestPgSaveBatchURL(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	urlPgStore := &URLPgStore{
		pool: mock,
	}

	tests := []struct {
		name   string
		dbErr  error
		resErr error
	}{
		{
			name: "Успешное сохранение",
		},
		{
			name:   "Ссылка уже была сохранена",
			dbErr:  &pgconn.PgError{Code: pgerrcode.UniqueViolation},
			resErr: ErrConflict,
		},
		{
			name:   "Ошибка БД",
			dbErr:  errors.New("db error"),
			resErr: fmt.Errorf("failed to save batch of urls: %w", errors.New("db error")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExpectBatch := mock.ExpectBatch().
				ExpectExec("INSERT INTO shorten_urls").
				WithArgs("some", pgxmock.AnyArg(), 1)
			if tt.dbErr != nil {
				mockExpectBatch.WillReturnError(tt.dbErr)
			} else {
				mockExpectBatch.WillReturnResult(pgxmock.NewResult("INSERT", 1))
			}

			storeErr := urlPgStore.SaveBatchURL(context.TODO(), []ShortenURL{{Original: "some"}}, 1)
			if tt.resErr != nil {
				assert.EqualError(t, tt.resErr, storeErr.Error())
			} else {
				require.NoError(t, storeErr)
			}
		})
	}
}

func TestPgGetUser(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	urlPgStore := &URLPgStore{
		pool: mock,
	}

	type dbRes struct {
		rows []any
		err  error
	}

	type want struct {
		user *User
		err  error
	}

	tests := []struct {
		name   string
		userID int
		dbRes  *dbRes
		want   want
	}{
		{
			name:   "Успешное чтение пользователя",
			userID: 1,
			dbRes: &dbRes{
				rows: []any{1},
			},
			want: want{
				user: &User{ID: 1},
			},
		},
		{
			name:   "Некорректный ID",
			userID: -1,
			want: want{
				err: errors.New("invalid user id"),
			},
		},
		{
			name:   "Не найден в БД",
			userID: 1,
			dbRes: &dbRes{
				err: pgx.ErrNoRows,
			},
			want: want{
				err: ErrNoData,
			},
		},
		{
			name:   "Ошибка БД",
			userID: 1,
			dbRes: &dbRes{
				err: errors.New("db error"),
			},
			want: want{
				err: fmt.Errorf("failed to select user from db: %w", errors.New("db error")),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.dbRes != nil {
				mockExpectQuery := mock.ExpectQuery("SELECT id FROM users").
					WithArgs(tt.userID)
				if tt.dbRes.err != nil {
					mockExpectQuery.WillReturnError(tt.dbRes.err)
				} else {
					mockExpectQuery.WillReturnRows(mock.NewRows([]string{"id"}).
						AddRow(tt.dbRes.rows...))
				}
			}

			user, storeErr := urlPgStore.GetUser(context.TODO(), tt.userID)
			if tt.want.err != nil {
				assert.EqualError(t, tt.want.err, storeErr.Error())
			} else {
				require.NoError(t, storeErr)
				assert.Equal(t, tt.want.user, user)
			}
		})
	}
}

func TestPgGetUserURLs(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	urlPgStore := &URLPgStore{
		pool: mock,
	}

	type want struct {
		urls []ShortenURL
		err  error
	}

	type dbRes struct {
		rows [][]any
		err  error
	}

	tests := []struct {
		name   string
		userID int
		dbRes  *dbRes
		want   want
	}{
		{
			name:   "Успешное чтение данных",
			userID: 1,
			dbRes: &dbRes{
				rows: [][]any{{"origin1", "shorten1"}, {"origin2", "shorten2"}},
			},
			want: want{
				urls: []ShortenURL{
					{Original: "origin1", Shorten: "shorten1"},
					{Original: "origin2", Shorten: "shorten2"},
				},
			},
		},
		{
			name:   "Некорректный ID",
			userID: -1,
			want: want{
				err: errors.New("invalid user id"),
			},
		},
		{
			name:   "Ошибка БД",
			userID: 1,
			dbRes: &dbRes{
				err: errors.New("db error"),
			},
			want: want{
				err: fmt.Errorf("failed to select user urls from db: %w", errors.New("db error")),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.dbRes != nil {
				mockExpectQuery := mock.ExpectQuery("SELECT .+ FROM shorten_urls WHERE .+").
					WithArgs(tt.userID)
				if tt.dbRes.err != nil {
					mockExpectQuery.WillReturnError(tt.dbRes.err)
				} else {
					mockExpectQuery.WillReturnRows(mock.NewRows([]string{"original", "shorten"}).
						AddRows(tt.dbRes.rows...))
				}
			}

			urls, storeErr := urlPgStore.GetUserURLs(context.TODO(), tt.userID)
			if tt.want.err != nil {
				assert.EqualError(t, tt.want.err, storeErr.Error())
			} else {
				require.NoError(t, storeErr)
				assert.Equal(t, tt.want.urls, urls)
			}
		})
	}
}

func TestPgIsValidID(t *testing.T) {
	urlPgStore := &URLPgStore{}

	tests := []struct {
		name    string
		id      string
		isValid bool
	}{
		{
			name:    "Валидный ID",
			id:      "abcd5678",
			isValid: true,
		},
		{
			name:    "Некорректная длина",
			id:      "abcd56",
			isValid: false,
		},
		{
			name:    "Некорректные символы",
			id:      "!,abcd56",
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := urlPgStore.IsValidID(tt.id)
			assert.Equal(t, tt.isValid, isValid)
		})
	}
}

func TestPgPing(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	urlPgStore := &URLPgStore{
		pool: mock,
	}

	mock.ExpectPing().WillReturnError(errors.New("db error"))
	storeErr := urlPgStore.Ping(context.TODO())
	require.Error(t, storeErr)

	mock.ExpectPing().WillDelayFor(0)
	storeErr = urlPgStore.Ping(context.TODO())
	require.NoError(t, storeErr)
}

func TestPgInitSchema(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS users").WillReturnResult(pgxmock.NewResult("CREATE TABLE", 0))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS shorten_urls").WillReturnResult(pgxmock.NewResult("CREATE TABLE", 0))
	mock.ExpectCommit()

	err = initSchema(context.TODO(), mock)
	require.NoError(t, err)
}

func TestPgExecuteDelBatch(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	urlPgStore := &URLPgStore{
		pool: mock,
	}

	mock.ExpectBatch().
		ExpectExec("UPDATE shorten_urls_qwer").
		WithArgs("shorten", 1).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1)).Times(1)

	delBatch := []urlDelBatchData{
		{
			urls:   []string{"short"},
			userID: 1,
		},
	}

	urlPgStore.executeDelBatch(context.TODO(), delBatch)
}

func TestPgGetUsersCount(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	urlPgStore := &URLPgStore{
		pool: mock,
	}

	type dbRes struct {
		rows []any
		err  error
	}

	type want struct {
		count int
		err   error
	}

	tests := []struct {
		name  string
		dbRes *dbRes
		want  want
	}{
		{
			name: "Успешный запрос",
			dbRes: &dbRes{
				rows: []any{1},
			},
			want: want{
				count: 1,
			},
		},
		{
			name: "Ошибка БД",
			dbRes: &dbRes{
				err: errors.New("db error"),
			},
			want: want{
				err: fmt.Errorf("failed to select users count from db: %w", errors.New("db error")),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.dbRes != nil {
				mockExpectQuery := mock.ExpectQuery("SELECT .+ FROM users;")
				if tt.dbRes.err != nil {
					mockExpectQuery.WillReturnError(tt.dbRes.err)
				} else if tt.dbRes.rows != nil {
					fmt.Println("going to return")
					mockExpectQuery.WillReturnRows(mock.NewRows([]string{"count"}).
						AddRow(tt.dbRes.rows...))
				}
			}
			count, storeErr := urlPgStore.GetUsersCount(context.TODO())
			if tt.want.err != nil {
				assert.Equal(t, tt.want.err, storeErr)
			} else {
				require.NoError(t, storeErr)
				assert.Equal(t, tt.want.count, count)
			}
		})
	}
}

func TestPgGetURLsCount(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	urlPgStore := &URLPgStore{
		pool: mock,
	}

	type dbRes struct {
		rows []any
		err  error
	}

	type want struct {
		count int
		err   error
	}

	tests := []struct {
		name  string
		dbRes *dbRes
		want  want
	}{
		{
			name: "Успешный запрос",
			dbRes: &dbRes{
				rows: []any{1},
			},
			want: want{
				count: 1,
			},
		},
		{
			name: "Ошибка БД",
			dbRes: &dbRes{
				err: errors.New("db error"),
			},
			want: want{
				err: fmt.Errorf("failed to select urls count from db: %w", errors.New("db error")),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.dbRes != nil {
				mockExpectQuery := mock.ExpectQuery("SELECT .+ FROM shorten_urls WHERE is_deleted = false;")
				if tt.dbRes.err != nil {
					mockExpectQuery.WillReturnError(tt.dbRes.err)
				} else if tt.dbRes.rows != nil {
					fmt.Println("going to return")
					mockExpectQuery.WillReturnRows(mock.NewRows([]string{"count"}).
						AddRow(tt.dbRes.rows...))
				}
			}
			count, storeErr := urlPgStore.GetURLsCount(context.TODO())
			if tt.want.err != nil {
				assert.Equal(t, tt.want.err, storeErr)
			} else {
				require.NoError(t, storeErr)
				assert.Equal(t, tt.want.count, count)
			}
		})
	}
}
