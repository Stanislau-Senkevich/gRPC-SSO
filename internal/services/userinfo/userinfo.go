package userinfo

import (
	"context"
	"fmt"
	"github.com/Stanislau-Senkevich/GRPC_SSO/internal/domain/models"
	grpcerror "github.com/Stanislau-Senkevich/GRPC_SSO/internal/error"
	jwtmanager "github.com/Stanislau-Senkevich/GRPC_SSO/internal/lib/jwt"
	"github.com/Stanislau-Senkevich/GRPC_SSO/internal/lib/sl"
	"github.com/Stanislau-Senkevich/GRPC_SSO/internal/repository"
	"golang.org/x/crypto/bcrypt"
	"log/slog"
)

type UserInfoService struct {
	log      *slog.Logger
	repo     repository.UserInfoRepository
	manager  *jwtmanager.Manager
	hashSalt string
}

// New creates and returns a new instance of the UserInfoService
func New(
	log *slog.Logger,
	repo repository.UserInfoRepository,
	manager *jwtmanager.Manager,
	hashSalt string,
) *UserInfoService {
	return &UserInfoService{
		log:      log,
		repo:     repo,
		manager:  manager,
		hashSalt: hashSalt,
	}
}

// GetUserInfo retrieves user information for the authenticated user making the request.
// It extracts the user ID from the context, then delegates the retrieval to the
// GetUserInfoByID method.
func (s *UserInfoService) GetUserInfo(ctx context.Context) (models.User, error) {
	userID, err := s.manager.GetUserIDFromContext(ctx)
	if err != nil {
		return models.User{}, err
	}

	return s.GetUserInfoByID(ctx, userID)
}

func (s *UserInfoService) GetUserInfoByID(ctx context.Context, userID int64) (models.User, error) {
	return s.repo.GetUserInfo(ctx, userID)
}

// UpdateUserInfo updates user information for the authenticated user making the request.
// It extracts the user ID from the context, then delegates the update operation to the
// UpdateUserInfo method of the underlying repository.
func (s *UserInfoService) UpdateUserInfo(
	ctx context.Context,
	updatedUser *models.User) error {
	userID, err := s.manager.GetUserIDFromContext(ctx)
	if err != nil {
		return err
	}

	return s.repo.UpdateUserInfo(ctx, userID, updatedUser)
}

// ChangePassword updates the password for the authenticated user making the request.
// It extracts the user ID from the context, combines the old and new passwords with
// the salt, generates a new password hash, and then delegates the password change
// operation to the ChangePassword method of the underlying repository.
func (s *UserInfoService) ChangePassword(
	ctx context.Context,
	oldPassword, newPassword string) error {
	const op = "userinfo.service.ChangePassword"

	log := s.log.With(
		slog.String("op", op),
	)

	newPasswordSalted := newPassword + s.hashSalt
	oldPasswordSalted := oldPassword + s.hashSalt

	passHash, err := bcrypt.GenerateFromPassword([]byte(newPasswordSalted), bcrypt.DefaultCost)
	if err != nil {
		log.Error("failed to generate password hash", sl.Err(err))
	}

	userID, err := s.manager.GetUserIDFromContext(ctx)
	if err != nil {
		return err
	}

	return s.repo.ChangePassword(ctx, userID, oldPasswordSalted, string(passHash))
}

func (s *UserInfoService) DeleteUser(ctx context.Context, userID int64) error {
	return s.repo.DeleteUser(ctx, userID)
}

func (s *UserInfoService) AddFamily(ctx context.Context, familyID int64, userID int64) error {
	const op = "userinfo.service.AddFamily"

	log := s.log.With(
		slog.String("op", op),
	)

	user, err := s.repo.GetUserInfo(ctx, userID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if isUserInFamily(user.FamilyIDs, familyID) {
		log.Warn(grpcerror.ErrUserInFamily.Error())
		return grpcerror.ErrUserInFamily
	}

	return s.repo.AddFamily(ctx, &user, familyID)
}

func (s *UserInfoService) DeleteFamily(ctx context.Context, familyID int64, userID int64) error {
	const op = "userinfo.service.DeleteFamily"

	log := s.log.With(
		slog.String("op", op),
	)

	user, err := s.repo.GetUserInfo(ctx, userID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if !isUserInFamily(user.FamilyIDs, familyID) {
		log.Warn(grpcerror.ErrUserNotInFamily.Error())
		return grpcerror.ErrUserNotInFamily
	}

	return s.repo.DeleteFamily(ctx, &user, familyID)
}

func isUserInFamily(families []int64, familyID int64) bool {
	for _, f := range families {
		if f == familyID {
			return true
		}
	}
	return false
}
