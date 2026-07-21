package auth

import (
	"errors"
	"strings"
	"time"

	"office-file-sharing/backend/internal/shared/middleware"
	"office-file-sharing/backend/internal/shared/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Service interface {
	Login(req LoginRequest) (*AuthResponse, error)
	Signup(req SignupRequest) (*AuthResponse, error)
}

type service struct {
	repo      Repository
	jwtSecret []byte
}

func NewService(repo Repository, jwtSecret []byte) Service {
	return &service{repo: repo, jwtSecret: jwtSecret}
}

func (s *service) Login(req LoginRequest) (*AuthResponse, error) {
	email := strings.TrimSpace(strings.ToLower(req.Email))
	user, err := s.repo.GetByEmail(email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	tokenString, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	user.PasswordHash = ""
	isAdmin := s.repo.CheckAdminAccess(user.Role, user.SchoolID)
	return &AuthResponse{
		Token:   tokenString,
		User:    *user,
		IsAdmin: isAdmin,
	}, nil
}

func (s *service) Signup(req SignupRequest) (*AuthResponse, error) {
	name := strings.TrimSpace(req.Name)
	email := strings.TrimSpace(strings.ToLower(req.Email))
	password := req.Password

	if name == "" || email == "" || len(password) < 8 {
		return nil, errors.New("name, email, and password (minimum 8 characters) are required")
	}

	if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return nil, errors.New("invalid email address format")
	}

	// Check if user already exists
	_, err := s.repo.GetByEmail(email)
	if err == nil {
		return nil, errors.New("user with this email already exists")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("failed to encrypt password")
	}

	var defaultSchool models.School
	var schoolID *uuid.UUID
	if err := s.repo.(*repository).db.First(&defaultSchool).Error; err == nil {
		schoolID = &defaultSchool.ID
	}

	newUser := &models.User{
		ID:           uuid.New(),
		Name:         name,
		Email:        email,
		PasswordHash: string(hash),
		SchoolID:     schoolID,
		Role:         "Student", // Default signup role
	}

	if err := s.repo.Create(newUser); err != nil {
		return nil, errors.New("failed to create user")
	}

	tokenString, err := s.generateToken(newUser)
	if err != nil {
		return nil, err
	}

	newUser.PasswordHash = ""
	isAdmin := s.repo.CheckAdminAccess(newUser.Role, newUser.SchoolID)
	return &AuthResponse{
		Token:   tokenString,
		User:    *newUser,
		IsAdmin: isAdmin,
	}, nil
}

func (s *service) generateToken(user *models.User) (string, error) {
	claims := &middleware.JWTCustomClaims{
		UserID: user.ID.String(),
		Email:  user.Email,
		Name:   user.Name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}
