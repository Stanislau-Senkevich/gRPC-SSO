package mongodb

import (
	"context"
	"fmt"
	"github.com/Stanislau-Senkevich/GRPC_SSO/internal/config"
	"github.com/Stanislau-Senkevich/GRPC_SSO/internal/domain/models"
	grpc_error "github.com/Stanislau-Senkevich/GRPC_SSO/internal/error"
	"github.com/Stanislau-Senkevich/GRPC_SSO/internal/lib/sl"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
	"log/slog"
)

// GetUserInfo retrieves user information for the user with the provided user ID
// from the MongoDB database. It returns the user object, excluding the password hash.
func (m *MongoRepository) GetUserInfo(ctx context.Context, userID int64) (models.User, error) {
	const op = "userinfo.mongo.GetUserInfo"

	var res models.User

	log := m.log.With(
		slog.String("op", op),
	)

	coll := m.Db.Database(m.Config.DBName).Collection(
		m.Config.Collections[config.UserCollection])

	filter := bson.D{
		{"user_id", userID},
	}

	singleRes := coll.FindOne(ctx, filter)
	if singleRes.Err() != nil {
		return models.User{}, grpc_error.ErrUserNotFound
	}

	if err := singleRes.Decode(&res); err != nil {
		log.Error("failed to decode user", sl.Err(err))
		return models.User{}, fmt.Errorf("failed to decode user: %w", err)
	}

	res.PassHash = ""

	return res, nil
}

// UpdateUserInfo updates user information for the user with the provided user ID
// in the MongoDB database. It first verifies the existence of the user, then merges
// the updatedUser fields with the existing user data, ensuring that non-updated
// fields are preserved.
func (m *MongoRepository) UpdateUserInfo(
	ctx context.Context,
	userID int64,
	updatedUser *models.User) error {
	const op = "userinfo.mongo.UpdateUserInfo"

	var user models.User

	log := m.log.With(
		slog.String("op", op),
	)

	coll := m.Db.Database(m.Config.DBName).Collection(
		m.Config.Collections[config.UserCollection])

	filter := bson.D{
		{"user_id", userID},
	}

	res := coll.FindOne(ctx, filter)
	if res.Err() != nil {
		return grpc_error.ErrUserNotFound
	}

	if err := res.Decode(&user); err != nil {
		log.Error("failed to decode user", sl.Err(err))
		return fmt.Errorf("failed to decode user: %w", err)
	}

	if updatedUser.Email == "" {
		updatedUser.Email = user.Email
	}

	checkUpdateInfo(&user, updatedUser)

	update := bson.M{
		"$set": bson.M{
			"email":        updatedUser.Email,
			"phone_number": updatedUser.PhoneNumber,
			"name":         updatedUser.Name,
			"surname":      updatedUser.Surname,
		},
	}

	if _, err := coll.UpdateOne(ctx, filter, update); err != nil {
		log.Error("failed to update user info", sl.Err(err))
		return fmt.Errorf("failed to update user info: %w", err)
	}

	return nil
}

// ChangePassword updates the password for the user with the provided user ID in
// the MongoDB database. It verifies the old password, and if successful, replaces
// it with the new password hash.
func (m *MongoRepository) ChangePassword(
	ctx context.Context,
	userID int64,
	oldPasswordSalted,
	newPasswordHash string) error {
	const op = "userinfo.mongo.ChangePassword"

	var user models.User

	log := m.log.With(
		slog.String("op", op),
	)

	coll := m.Db.Database(m.Config.DBName).Collection(
		m.Config.Collections[config.UserCollection])

	filter := bson.D{
		{"user_id", userID},
	}

	res := coll.FindOne(ctx, filter)
	if res.Err() != nil {
		return grpc_error.ErrUserNotFound
	}

	if err := res.Decode(&user); err != nil {
		log.Error("failed to decode user", sl.Err(err))
		return fmt.Errorf("failed to decode user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PassHash), []byte(oldPasswordSalted)); err != nil {
		log.Info(grpc_error.ErrInvalidPassword.Error(), slog.Int64("user_id", userID))
		return grpc_error.ErrInvalidPassword
	}

	update := bson.M{
		"$set": bson.M{"pass_hash": newPasswordHash},
	}

	if _, err := coll.UpdateOne(ctx, filter, update); err != nil {
		log.Error("failed to change password", sl.Err(err))
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// DeleteUser removes a user from the MongoDB database using the provided user ID.
// It first checks if the user exists, and if found, deletes the user from the database.
func (m *MongoRepository) DeleteUser(ctx context.Context, userID int64) error {
	const op = "userinfo.mongo.DeleteUser"

	log := m.log.With(
		slog.String("op", op),
	)

	coll := m.Db.Database(m.Config.DBName).Collection(
		m.Config.Collections[config.UserCollection])

	filter := bson.D{
		{"user_id", userID},
	}

	if res := coll.FindOne(ctx, filter); res.Err() != nil {
		return grpc_error.ErrUserNotFound
	}

	if _, err := coll.DeleteOne(ctx, filter); err != nil {
		log.Error("failed to delete user", sl.Err(err), slog.Int64("user_id", userID))
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// checkUpdateInfo ensures that the provided updateInfo object contains valid
// information for updating a user. If any field in updateInfo is empty, it is
// replaced with the corresponding field from the original user object.
func checkUpdateInfo(user, updateInfo *models.User) {
	if updateInfo.Email == "" {
		updateInfo.Email = user.Email
	}

	if updateInfo.Name == "" {
		updateInfo.Name = user.Name
	}

	if updateInfo.Surname == "" {
		updateInfo.Surname = user.Surname
	}

	if updateInfo.PhoneNumber == "" {
		updateInfo.PhoneNumber = user.PhoneNumber
	}
}