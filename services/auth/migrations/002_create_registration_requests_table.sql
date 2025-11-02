-- CreateUser registration_requests table
CREATE TABLE IF NOT EXISTS registration_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    password TEXT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    approved_by UUID,
    approved_at TIMESTAMP,
    CONSTRAINT fk_approved_by FOREIGN KEY (approved_by) REFERENCES users(id) ON DELETE SET NULL
);

-- CreateUser index on status for faster filtering
CREATE INDEX idx_registration_requests_status ON registration_requests(status);

-- CreateUser index on email for faster lookups
CREATE INDEX idx_registration_requests_email ON registration_requests(email);

-- CreateUser index on created_at for sorting
CREATE INDEX idx_registration_requests_created_at ON registration_requests(created_at DESC);
