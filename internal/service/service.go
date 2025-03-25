package service

import test "serviceLyceum/pkg/api/test/api"

type Service struct {
	test.OrderServiceServer
}

func NewService() *Service {
	return &Service{}
}
