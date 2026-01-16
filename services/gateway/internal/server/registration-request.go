package server

import "github.com/gofiber/fiber/v2"

func (s *Server) setupRegistrationRequestRoutes() {
	registrationRequests := s.router.Group("/api/v1/registration-requests")
	registrationRequests.Get("/", s.listRegistrationRequests)
	registrationRequests.Post("/:id/approve", s.approveRegistrationRequest)
	registrationRequests.Post("/:id/reject", s.rejectRegistrationRequest)
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
