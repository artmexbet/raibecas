package server

import (
	"github.com/artmexbet/raibecas/libs/natsw"

	"github.com/artmexbet/raibecas/services/users/internal/handler"
)

type Server struct {
	client  *natsw.Client
	handler *handler.Handler
}

func New(client *natsw.Client, handler *handler.Handler) Server {
	return Server{
		client:  client,
		handler: handler,
	}
}

func (s *Server) Start() error {
	s.client.Subscribe("users.list", s.handler.HandleListUsers)
	s.client.Subscribe("users.get", s.handler.HandleGetUser)
	s.client.Subscribe("users.update", s.handler.HandleUpdateUser)
	s.client.Subscribe("users.delete", s.handler.HandleDeleteUser)

	s.client.Subscribe("users.registration.create", s.handler.HandleCreateRegistration)
	s.client.Subscribe("users.registration.list", s.handler.HandleListRegistrations)
	s.client.Subscribe("users.registration.approve", s.handler.HandleApproveRegistration)
	s.client.Subscribe("users.registration.reject", s.handler.HandleRejectRegistration)

	return nil
}
