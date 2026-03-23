package books

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrInvalidBookID は書籍 ID の形式や値が不正なときに返す。
var ErrInvalidBookID = errors.New("invalid book id")

// ErrInvalidBookInput は書籍の入力値が不正なときに返す。
var ErrInvalidBookInput = errors.New("invalid book input")

// ErrBookNotFound は対象書籍が存在しないときに返す。
var ErrBookNotFound = errors.New("book not found")

// ErrBookDeleteRestricted は関連データがあるため削除できないときに返す。
var ErrBookDeleteRestricted = errors.New("book delete restricted")

type dbQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// Service は books テーブルに対する CRUD を提供する。
type Service struct {
	db dbQuerier
}

// Book は API 応答に返す書誌情報。
type Book struct {
	BookID        string  `json:"id"`
	Title         string  `json:"title"`
	Author        string  `json:"author"`
	ISBN          *string `json:"isbn"`
	Publisher     *string `json:"publisher"`
	PublishedYear *int    `json:"published_year"`
	CategoryID    string  `json:"category_id"`
	Location      *string `json:"location"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

// CreateBookInput は蔵書作成時の入力値。
type CreateBookInput struct {
	Title         string
	Author        string
	ISBN          *string
	Publisher     *string
	PublishedYear *int
	CategoryID    string
	Location      *string
}

// UpdateBookInput は蔵書更新時の入力値。
type UpdateBookInput struct {
	BookID        string
	Title         string
	Author        string
	ISBN          *string
	Publisher     *string
	PublishedYear *int
	CategoryID    string
	Location      *string
}

// NewService は書籍 CRUD サービスを生成する。
func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

// newService はテスト差し替え用の DB 依存を受け取る。
func newService(db dbQuerier) *Service {
	return &Service{db: db}
}

// CreateBook は books テーブルへ新規書籍を登録する。
func (s *Service) CreateBook(ctx context.Context, in CreateBookInput) (Book, error) {
	normalized, err := normalizeBookInput(
		strings.TrimSpace(in.Title),
		strings.TrimSpace(in.Author),
		strings.TrimSpace(in.CategoryID),
		in.ISBN,
		in.Publisher,
		in.PublishedYear,
		in.Location,
	)
	if err != nil {
		return Book{}, err
	}

	row := s.db.QueryRow(ctx, `
		INSERT INTO books (title, author, isbn, publisher, published_year, category_id, location, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6::uuid, $7, NOW(), NOW())
		RETURNING id::text, title, author, isbn, publisher, published_year, category_id::text, location, created_at, updated_at
	`, normalized.title, normalized.author, normalized.isbn, normalized.publisher, normalized.publishedYear, normalized.categoryID, normalized.location)

	book, scanErr := scanBook(row)
	if scanErr != nil {
		return Book{}, mapWriteError(scanErr)
	}

	return book, nil
}

// ListBooks は管理画面表示向けに書籍一覧を返す。
func (s *Service) ListBooks(ctx context.Context) ([]Book, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id::text, title, author, isbn, publisher, published_year, category_id::text, location, created_at, updated_at
		FROM books
		ORDER BY created_at DESC, id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	books := make([]Book, 0)
	for rows.Next() {
		book, scanErr := scanBook(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		books = append(books, book)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return books, nil
}

// GetBook は book_id で書籍詳細を取得する。
func (s *Service) GetBook(ctx context.Context, bookID string) (Book, error) {
	normalizedBookID := strings.TrimSpace(bookID)
	if normalizedBookID == "" {
		return Book{}, ErrInvalidBookID
	}

	row := s.db.QueryRow(ctx, `
		SELECT id::text, title, author, isbn, publisher, published_year, category_id::text, location, created_at, updated_at
		FROM books
		WHERE id = $1::uuid
		LIMIT 1
	`, normalizedBookID)

	book, err := scanBook(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Book{}, ErrBookNotFound
		}
		if isInvalidUUIDError(err) {
			return Book{}, ErrInvalidBookID
		}
		return Book{}, err
	}

	return book, nil
}

// UpdateBook は指定書籍の書誌情報を更新する。
func (s *Service) UpdateBook(ctx context.Context, in UpdateBookInput) (Book, error) {
	normalizedBookID := strings.TrimSpace(in.BookID)
	if normalizedBookID == "" {
		return Book{}, ErrInvalidBookID
	}

	normalized, err := normalizeBookInput(
		strings.TrimSpace(in.Title),
		strings.TrimSpace(in.Author),
		strings.TrimSpace(in.CategoryID),
		in.ISBN,
		in.Publisher,
		in.PublishedYear,
		in.Location,
	)
	if err != nil {
		return Book{}, err
	}

	row := s.db.QueryRow(ctx, `
		UPDATE books
		SET title = $2,
		    author = $3,
		    isbn = $4,
		    publisher = $5,
		    published_year = $6,
		    category_id = $7::uuid,
		    location = $8,
		    updated_at = NOW()
		WHERE id = $1::uuid
		RETURNING id::text, title, author, isbn, publisher, published_year, category_id::text, location, created_at, updated_at
	`, normalizedBookID, normalized.title, normalized.author, normalized.isbn, normalized.publisher, normalized.publishedYear, normalized.categoryID, normalized.location)

	book, scanErr := scanBook(row)
	if scanErr != nil {
		if errors.Is(scanErr, pgx.ErrNoRows) {
			return Book{}, ErrBookNotFound
		}
		if isInvalidUUIDError(scanErr) {
			return Book{}, ErrInvalidBookID
		}
		return Book{}, mapWriteError(scanErr)
	}

	return book, nil
}

// DeleteBook は指定書籍を削除する。
func (s *Service) DeleteBook(ctx context.Context, bookID string) error {
	normalizedBookID := strings.TrimSpace(bookID)
	if normalizedBookID == "" {
		return ErrInvalidBookID
	}

	hasReservations, err := s.bookHasReservations(ctx, normalizedBookID)
	if err != nil {
		return err
	}
	// 予約履歴は履歴参照要件の対象なので、書籍削除で消さない。
	if hasReservations {
		return ErrBookDeleteRestricted
	}

	var deletedBookID string
	err = s.db.QueryRow(ctx, `
		DELETE FROM books
		WHERE id = $1::uuid
		RETURNING id::text
	`, normalizedBookID).Scan(&deletedBookID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrBookNotFound
		}
		if isInvalidUUIDError(err) {
			return ErrInvalidBookID
		}
		if isForeignKeyViolation(err) {
			return ErrBookDeleteRestricted
		}
		return err
	}

	return nil
}

// bookHasReservations は対象書籍に予約履歴が存在するかを返す。
func (s *Service) bookHasReservations(ctx context.Context, bookID string) (bool, error) {
	var hasReservations bool
	err := s.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM reservations
			WHERE book_id = $1::uuid
		)
	`, bookID).Scan(&hasReservations)
	if err != nil {
		if isInvalidUUIDError(err) {
			return false, ErrInvalidBookID
		}
		return false, err
	}
	return hasReservations, nil
}

type normalizedBookInput struct {
	title         string
	author        string
	isbn          any
	publisher     any
	publishedYear any
	categoryID    string
	location      any
}

// normalizeBookInput は入力値の trim と任意項目の NULL 変換を行う。
func normalizeBookInput(title, author, categoryID string, isbn, publisher *string, publishedYear *int, location *string) (normalizedBookInput, error) {
	if title == "" || author == "" || categoryID == "" {
		return normalizedBookInput{}, ErrInvalidBookInput
	}

	return normalizedBookInput{
		title:         title,
		author:        author,
		isbn:          optionalStringParam(isbn),
		publisher:     optionalStringParam(publisher),
		publishedYear: optionalIntParam(publishedYear),
		categoryID:    categoryID,
		location:      optionalStringParam(location),
	}, nil
}

// optionalStringParam は空文字を NULL に寄せて DB 保存値を揃える。
func optionalStringParam(value *string) any {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

// optionalIntParam は nil を NULL として扱う。
func optionalIntParam(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

// scanBook は SELECT/RETURNING の1行を Book 構造体へ変換する。
func scanBook(row interface{ Scan(dest ...any) error }) (Book, error) {
	var (
		book          Book
		isbn          sql.NullString
		publisher     sql.NullString
		publishedYear sql.NullInt32
		location      sql.NullString
		createdAt     time.Time
		updatedAt     time.Time
	)

	err := row.Scan(
		&book.BookID,
		&book.Title,
		&book.Author,
		&isbn,
		&publisher,
		&publishedYear,
		&book.CategoryID,
		&location,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return Book{}, err
	}

	book.ISBN = nullStringPtr(isbn)
	book.Publisher = nullStringPtr(publisher)
	book.PublishedYear = nullIntPtr(publishedYear)
	book.Location = nullStringPtr(location)
	book.CreatedAt = createdAt.Format(time.RFC3339)
	book.UpdatedAt = updatedAt.Format(time.RFC3339)

	return book, nil
}

// nullStringPtr は NULL 許容文字列をポインタへ変換する。
func nullStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	v := value.String
	return &v
}

// nullIntPtr は NULL 許容整数をポインタへ変換する。
func nullIntPtr(value sql.NullInt32) *int {
	if !value.Valid {
		return nil
	}
	v := int(value.Int32)
	return &v
}

// mapWriteError は更新系 SQL の DB 例外を業務エラーへ寄せる。
func mapWriteError(err error) error {
	if isInvalidInputError(err) {
		return ErrInvalidBookInput
	}
	return err
}

// isInvalidInputError は FK/CHECK/UUID など入力起因の DB エラーを判定する。
func isInvalidInputError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "22P02", "23503", "23514", "23502":
			return true
		}
	}
	return false
}

// isInvalidUUIDError は UUID キャスト失敗（22P02）を判定する。
func isInvalidUUIDError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "22P02"
	}
	return false
}

// isForeignKeyViolation は外部キー制約違反（23503）を判定する。
func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23503"
	}
	return false
}
