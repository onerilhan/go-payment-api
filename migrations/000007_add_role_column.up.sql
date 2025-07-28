-- Add role column to users table
ALTER TABLE users 
ADD COLUMN role VARCHAR(20) DEFAULT 'user' NOT NULL CHECK (role IN ('user', 'admin', 'mod'));

-- Create index for role queries
CREATE INDEX idx_users_role ON users(role);

-- Update existing users to have 'user' role (if any)
UPDATE users SET role = 'user' WHERE role IS NULL;