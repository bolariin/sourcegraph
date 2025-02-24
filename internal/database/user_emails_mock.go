package database

import (
	"context"
)

type MockUserEmails struct {
	GetPrimaryEmail                func(ctx context.Context, id int32) (email string, verified bool, err error)
	Get                            func(userID int32, email string) (emailCanonicalCase string, verified bool, err error)
	SetPrimaryEmail                func(ctx context.Context, userID int32, email string) error
	SetVerified                    func(ctx context.Context, userID int32, email string, verified bool) error
	SetLastVerification            func(ctx context.Context, userID int32, email, code string) error
	GetLatestVerificationSentEmail func(ctx context.Context, email string) (*UserEmail, error)
	GetVerifiedEmails              func(ctx context.Context, emails ...string) ([]*UserEmail, error)
	ListByUser                     func(ctx context.Context, opt UserEmailsListOptions) ([]*UserEmail, error)
	Verify                         func(ctx context.Context, userID int32, email, code string) (bool, error)
}
