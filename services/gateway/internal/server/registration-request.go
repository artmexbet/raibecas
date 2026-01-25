package server

import "github.com/gofiber/fiber/v2"

func (s *Server) createRegistrationRequest(c *fiber.Ctx) error {
	return c.SendString("create registration request")
}

func (s *Server) listRegistrationRequests(c *fiber.Ctx) error {
	return c.SendString("list registration requests")
}

func (s *Server) approveRegistrationRequest(c *fiber.Ctx) error {
	return c.SendString("approve registration request")
}

func (s *Server) rejectRegistrationRequest(c *fiber.Ctx) error {
	return c.SendString("reject registration request")
}
