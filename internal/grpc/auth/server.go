package auth

import (
	"context"
	"errors"
	"gRPC_auth/internal/services/auth"
	"gRPC_auth/internal/storage"
	"gRPC_auth/protos/gen/go/sso"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Auth interface {
	Login(ctx context.Context,
		email string,
		password string,
		appID int,
	) (token string, err error)

	RegisterNewUser(ctx context.Context,
		email string,
		password string,
	) (userID int64, err error)

	IsAdmin(ctx context.Context, userID int64) (bool, error)
}

const (
	emptyID    = 0
	emptyValue = ""
)

type serverAPI struct {
	sso.UnimplementedAuthServer
	auth Auth
}

func Register(gRPC *grpc.Server, auth Auth) {
	sso.RegisterAuthServer(gRPC, &serverAPI{auth: auth})
}

func (s *serverAPI) Login(ctx context.Context, req *sso.LoginRequest) (*sso.LoginResponse, error) {
	if err := validateLogin(req); err != nil {
		return nil, err
	}

	token, err := s.auth.Login(ctx, req.GetEmail(), req.GetPassword(), int(req.GetAppId()))
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			return nil, status.Errorf(codes.InvalidArgument, "invalid credentials")
		}
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &sso.LoginResponse{
		Token: token,
	}, nil
}

func (s *serverAPI) Register(ctx context.Context, req *sso.RegisterRequest) (*sso.RegisterResponse, error) {
	if err := validateRegister(req); err != nil {
		return nil, err
	}

	userID, err := s.auth.RegisterNewUser(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		if errors.Is(err, storage.ErrUserExist) {
			return nil, status.Errorf(codes.AlreadyExists, "user already exists")
		}

		return nil, status.Error(codes.Internal, "internal error")
	}

	return &sso.RegisterResponse{
		UserId: userID,
	}, nil
}

func (s *serverAPI) IsAdmin(ctx context.Context, req *sso.IsAdminRequest) (*sso.IsAdminResponse, error) {
	if err := validateIsAdmin(req); err != nil {
		return nil, err
	}

	isAdmin, err := s.auth.IsAdmin(ctx, req.GetUserId())
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return nil, status.Errorf(codes.NotFound, "user not found")
		}

		return nil, status.Error(codes.Internal, "internal error")
	}

	return &sso.IsAdminResponse{
		IsAdmin: isAdmin,
	}, nil
}

func validateLogin(req *sso.LoginRequest) error {
	if req.GetEmail() == emptyValue {
		return status.Error(codes.InvalidArgument, "email is required")
	}

	if req.GetPassword() == emptyValue {
		return status.Error(codes.InvalidArgument, "password is required")
	}

	if req.GetAppId() == emptyID {
		return status.Error(codes.InvalidArgument, "app_id is required")
	}

	return nil
}

func validateRegister(req *sso.RegisterRequest) error {
	if req.GetEmail() == emptyValue {
		return status.Error(codes.InvalidArgument, "email is required")
	}

	if req.GetPassword() == emptyValue {
		return status.Error(codes.InvalidArgument, "password is required")
	}

	return nil
}

func validateIsAdmin(req *sso.IsAdminRequest) error {
	if req.GetUserId() == emptyID {
		return status.Error(codes.InvalidArgument, "user_id is required")
	}

	return nil
}
