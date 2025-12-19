package service

import (
	"context"

	pb "yinni_backend/api/user/v1"
	"yinni_backend/app/user/internal/biz"
)

type UserService struct {
	pb.UnimplementedUserServer

	uc *biz.UserUsecase
}

func NewUserService(uc *biz.UserUsecase) *UserService {
	return &UserService{uc: uc}
}

func (s *UserService) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserReply, error) {
	user, err := s.uc.CreateUser(ctx, &biz.User{
		Name:  req.Name,
		Email: req.Email,
		Age:   int(req.Age),
	})

	if err != nil {
		return nil, err
	}

	return &pb.CreateUserReply{Id: user.ID}, nil
}

func (s *UserService) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserReply, error) {
	user, err := s.uc.UpdateUser(ctx, &biz.User{
		Name:  req.Name,
		Email: req.Email,
		Age:   int(req.Age),
	})

	if err != nil {
		return nil, err
	}

	return &pb.UpdateUserReply{Id: user.ID}, nil
}

func (s *UserService) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserReply, error) {
	user, err := s.uc.DeleteUser(ctx, req.Id)

	if err != nil {
		return nil, err
	}

	return &pb.DeleteUserReply{Id: user.ID}, nil
}

func (s *UserService) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserReply, error) {
	user, err := s.uc.GetUser(ctx, req.Id)

	if err != nil {
		return nil, err
	}

	return &pb.GetUserReply{Id: user.ID, Name: user.Name, Email: user.Email, Age: int32(user.Age)}, nil
}

func (s *UserService) ListUser(ctx context.Context, req *pb.ListUserRequest) (*pb.ListUserReply, error) {
	users, err := s.uc.ListAllUser(ctx)

	if err != nil {
		return nil, err
	}

	reply := &pb.ListUserReply{}
	for _, user := range users {
		reply.Results = append(reply.Results, &pb.GetUserReply{
			Id:    user.ID,
			Name:  user.Name,
			Email: user.Email,
			Age:   int32(user.Age),
		})
	}

	return reply, nil
}
