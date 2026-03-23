package books

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestCreateBook(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	title := "Go in Action"
	author := "William Kennedy"
	isbn := "9781617291784"
	publisher := "Manning"
	publishedYear := 2016
	location := "A-1"
	categoryID := "8eb4f46c-4706-4a22-80df-33576f32f7f8"

	svc := newService(fakeDB{
		queryRowFunc: func(_ context.Context, _ string, args ...any) pgx.Row {
			if got, want := args[0], title; got != want {
				t.Fatalf("title = %v, want %v", got, want)
			}
			if got, want := args[1], author; got != want {
				t.Fatalf("author = %v, want %v", got, want)
			}
			if got, want := args[2], isbn; got != want {
				t.Fatalf("isbn = %v, want %v", got, want)
			}
			if got, want := args[5], categoryID; got != want {
				t.Fatalf("category_id = %v, want %v", got, want)
			}
			return fakeRow{values: []any{
				"1a302fd3-95ce-4bc8-989f-cdfac212f9e7",
				title,
				author,
				sql.NullString{String: isbn, Valid: true},
				sql.NullString{String: publisher, Valid: true},
				sql.NullInt32{Int32: int32(publishedYear), Valid: true},
				categoryID,
				sql.NullString{String: location, Valid: true},
				now,
				now,
			}}
		},
	})

	book, err := svc.CreateBook(context.Background(), CreateBookInput{
		Title:         title,
		Author:        author,
		ISBN:          &isbn,
		Publisher:     &publisher,
		PublishedYear: &publishedYear,
		CategoryID:    categoryID,
		Location:      &location,
	})
	if err != nil {
		t.Fatalf("CreateBook() error = %v", err)
	}

	if book.Title != title {
		t.Fatalf("book.Title = %q, want %q", book.Title, title)
	}
	if book.ISBN == nil || *book.ISBN != isbn {
		t.Fatalf("book.ISBN = %v, want %q", book.ISBN, isbn)
	}
}

func TestCreateBookRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	svc := newService(fakeDB{})

	_, err := svc.CreateBook(context.Background(), CreateBookInput{
		Title:      "",
		Author:     "author",
		CategoryID: "de9d2de8-e174-425a-a9e4-95ef0ce9fd61",
	})
	if err != ErrInvalidBookInput {
		t.Fatalf("CreateBook() error = %v, want %v", err, ErrInvalidBookInput)
	}
}

func TestListBooks(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	svc := newService(fakeDB{
		queryFunc: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			return &fakeRows{values: [][]any{
				{
					"book-1",
					"Book 1",
					"Author 1",
					sql.NullString{String: "isbn-1", Valid: true},
					sql.NullString{Valid: false},
					sql.NullInt32{Valid: false},
					"category-1",
					sql.NullString{Valid: false},
					now,
					now,
				},
			}}, nil
		},
	})

	books, err := svc.ListBooks(context.Background())
	if err != nil {
		t.Fatalf("ListBooks() error = %v", err)
	}
	if len(books) != 1 {
		t.Fatalf("books count = %d, want 1", len(books))
	}
	if books[0].ISBN == nil || *books[0].ISBN != "isbn-1" {
		t.Fatalf("books[0].ISBN = %v", books[0].ISBN)
	}
}

func TestGetBookReturnsNotFound(t *testing.T) {
	t.Parallel()

	svc := newService(fakeDB{
		queryRowFunc: func(_ context.Context, _ string, _ ...any) pgx.Row {
			return fakeRow{err: pgx.ErrNoRows}
		},
	})

	_, err := svc.GetBook(context.Background(), "088ab2a8-f105-4186-a2f5-4100a6d4d808")
	if err != ErrBookNotFound {
		t.Fatalf("GetBook() error = %v, want %v", err, ErrBookNotFound)
	}
}

func TestUpdateBookReturnsInvalidBookID(t *testing.T) {
	t.Parallel()

	svc := newService(fakeDB{})

	_, err := svc.UpdateBook(context.Background(), UpdateBookInput{
		BookID:     "",
		Title:      "book",
		Author:     "author",
		CategoryID: "12ce8666-f60f-4be7-b8a6-f98996b0c3ad",
	})
	if err != ErrInvalidBookID {
		t.Fatalf("UpdateBook() error = %v, want %v", err, ErrInvalidBookID)
	}
}

func TestDeleteBook(t *testing.T) {
	t.Parallel()

	svc := newService(fakeDB{
		queryRowFunc: func(_ context.Context, _ string, args ...any) pgx.Row {
			if got, want := args[0], "4db4f287-05fd-4d9e-90de-10c4bfd37d3f"; got != want {
				t.Fatalf("book id = %v, want %v", got, want)
			}
			return fakeRow{values: []any{"4db4f287-05fd-4d9e-90de-10c4bfd37d3f"}}
		},
	})

	err := svc.DeleteBook(context.Background(), "4db4f287-05fd-4d9e-90de-10c4bfd37d3f")
	if err != nil {
		t.Fatalf("DeleteBook() error = %v", err)
	}
}

func TestDeleteBookRejectsInvalidUUID(t *testing.T) {
	t.Parallel()

	svc := newService(fakeDB{
		queryRowFunc: func(_ context.Context, _ string, _ ...any) pgx.Row {
			return fakeRow{err: &pgconn.PgError{Code: "22P02"}}
		},
	})

	err := svc.DeleteBook(context.Background(), "invalid")
	if err != ErrInvalidBookID {
		t.Fatalf("DeleteBook() error = %v, want %v", err, ErrInvalidBookID)
	}
}

type fakeDB struct {
	queryRowFunc func(ctx context.Context, sql string, args ...any) pgx.Row
	queryFunc    func(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func (db fakeDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if db.queryRowFunc == nil {
		return fakeRow{err: errors.New("unexpected query row")}
	}
	return db.queryRowFunc(ctx, sql, args...)
}

func (db fakeDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if db.queryFunc == nil {
		return nil, errors.New("unexpected query")
	}
	return db.queryFunc(ctx, sql, args...)
}

type fakeRow struct {
	values []any
	err    error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i := range dest {
		switch d := dest[i].(type) {
		case *string:
			*d = r.values[i].(string)
		case *sql.NullString:
			*d = r.values[i].(sql.NullString)
		case *sql.NullInt32:
			*d = r.values[i].(sql.NullInt32)
		case *time.Time:
			*d = r.values[i].(time.Time)
		default:
			return errors.New("unsupported scan destination")
		}
	}
	return nil
}

type fakeRows struct {
	values [][]any
	idx    int
	err    error
}

func (r *fakeRows) Close() {}

func (r *fakeRows) Err() error {
	return r.err
}

func (r *fakeRows) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}

func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (r *fakeRows) Next() bool {
	if r.idx >= len(r.values) {
		return false
	}
	r.idx++
	return true
}

func (r *fakeRows) Scan(dest ...any) error {
	if r.idx == 0 || r.idx > len(r.values) {
		return errors.New("invalid row index")
	}
	current := r.values[r.idx-1]
	for i := range dest {
		switch d := dest[i].(type) {
		case *string:
			*d = current[i].(string)
		case *sql.NullString:
			*d = current[i].(sql.NullString)
		case *sql.NullInt32:
			*d = current[i].(sql.NullInt32)
		case *time.Time:
			*d = current[i].(time.Time)
		default:
			return errors.New("unsupported scan destination")
		}
	}
	return nil
}

func (r *fakeRows) Values() ([]any, error) {
	if r.idx == 0 || r.idx > len(r.values) {
		return nil, errors.New("invalid row index")
	}
	return r.values[r.idx-1], nil
}

func (r *fakeRows) RawValues() [][]byte {
	return nil
}

func (r *fakeRows) Conn() *pgx.Conn {
	return nil
}
