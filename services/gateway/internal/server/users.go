package server

import "github.com/gofiber/fiber/v2"

func (s *Server) setupUsersRoutes() {
	users := s.router.Group("/api/v1/users")
	users.Get("/", s.listUsers)
	users.Patch("/:id", s.updateUser)
	users.Delete("/:id", s.deleteUser)
}

func (s *Server) listUsers(c *fiber.Ctx) error {
	return c.SendString("list users")
}

func (s *Server) updateUser(c *fiber.Ctx) error {
	return c.SendString("update user request")
}

func (s *Server) deleteUser(c *fiber.Ctx) error {
	return c.SendString("delete user request")
}
