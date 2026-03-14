-- LibraryHub MVP schema (T-002)
-- PostgreSQL 16

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TYPE role_type AS ENUM ('ADMIN', 'USER');
CREATE TYPE user_status_type AS ENUM ('ACTIVE', 'INACTIVE');
CREATE TYPE copy_status_type AS ENUM ('AVAILABLE', 'ON_LOAN', 'RESERVED', 'INACTIVE');
CREATE TYPE loan_status_type AS ENUM ('ON_LOAN', 'OVERDUE', 'RETURNED');
CREATE TYPE reservation_status_type AS ENUM ('WAITING', 'NOTIFIED', 'FULFILLED', 'CANCELED');
CREATE TYPE notification_status_type AS ENUM ('PENDING', 'SENT', 'READ', 'FAILED');
CREATE TYPE notification_type AS ENUM ('DUE_SOON', 'OVERDUE_REMINDER', 'RESERVATION_AVAILABLE', 'SYSTEM');

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id VARCHAR(64) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role role_type NOT NULL DEFAULT 'USER',
    status user_status_type NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(120) NOT NULL UNIQUE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE books (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    author VARCHAR(255) NOT NULL,
    isbn VARCHAR(32),
    publisher VARCHAR(255),
    published_year INTEGER,
    category_id UUID NOT NULL REFERENCES categories(id),
    location VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT books_published_year_check CHECK (
        published_year IS NULL OR (published_year BETWEEN 1000 AND 2999)
    )
);

CREATE TABLE book_copies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    book_id UUID NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    copy_code VARCHAR(64) NOT NULL UNIQUE,
    status copy_status_type NOT NULL DEFAULT 'AVAILABLE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE loans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    book_copy_id UUID NOT NULL REFERENCES book_copies(id),
    loaned_at TIMESTAMPTZ NOT NULL,
    due_at TIMESTAMPTZ NOT NULL,
    returned_at TIMESTAMPTZ,
    extension_count INTEGER NOT NULL DEFAULT 0,
    status loan_status_type NOT NULL DEFAULT 'ON_LOAN',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT loans_due_after_loan_check CHECK (due_at > loaned_at),
    CONSTRAINT loans_extension_non_negative_check CHECK (extension_count >= 0 AND extension_count <= 1)
);

CREATE TABLE reservations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    book_id UUID NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    priority INTEGER NOT NULL,
    status reservation_status_type NOT NULL DEFAULT 'WAITING',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    notified_at TIMESTAMPTZ,
    canceled_at TIMESTAMPTZ,
    fulfilled_at TIMESTAMPTZ,
    CONSTRAINT reservations_priority_positive_check CHECK (priority > 0)
);

CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    type notification_type NOT NULL,
    message TEXT NOT NULL,
    scheduled_at TIMESTAMPTZ NOT NULL,
    sent_at TIMESTAMPTZ,
    read_at TIMESTAMPTZ,
    status notification_status_type NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT notifications_sent_after_scheduled_check CHECK (
        sent_at IS NULL OR sent_at >= scheduled_at
    )
);

CREATE TABLE admin_picks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    book_id UUID NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    start_at TIMESTAMPTZ NOT NULL,
    end_at TIMESTAMPTZ NOT NULL,
    priority INTEGER NOT NULL DEFAULT 100,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT admin_picks_period_check CHECK (end_at > start_at)
);

CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_user_id UUID REFERENCES users(id),
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    resource_id UUID,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for search/list performance
CREATE INDEX idx_books_title ON books (title);
CREATE INDEX idx_books_author ON books (author);
CREATE INDEX idx_books_isbn ON books (isbn);
CREATE INDEX idx_books_category_id ON books (category_id);

CREATE INDEX idx_book_copies_book_id_status ON book_copies (book_id, status);

CREATE INDEX idx_loans_user_id_status ON loans (user_id, status);
CREATE INDEX idx_loans_due_at_status ON loans (due_at, status);
CREATE INDEX idx_loans_book_copy_id_status ON loans (book_copy_id, status);

CREATE INDEX idx_reservations_book_id_status_priority ON reservations (book_id, status, priority);
CREATE INDEX idx_reservations_user_id_status ON reservations (user_id, status);

CREATE INDEX idx_notifications_user_id_status ON notifications (user_id, status);
CREATE INDEX idx_notifications_scheduled_at_status ON notifications (scheduled_at, status);

CREATE INDEX idx_admin_picks_active_period ON admin_picks (is_active, start_at, end_at);

CREATE INDEX idx_audit_logs_created_at ON audit_logs (created_at);
CREATE INDEX idx_audit_logs_actor_user_id_created_at ON audit_logs (actor_user_id, created_at);
