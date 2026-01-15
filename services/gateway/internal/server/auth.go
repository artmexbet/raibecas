package server

import "github.com/gofiber/fiber/v2"

func (s *Server) setupAuthRoutes() {
	auth := s.router.Group("/auth")
	auth.Post("/login", s.login)
	auth.Post("/logout", s.logout)
	auth.Post("/refresh-token", s.refreshToken)
	auth.Get("/me", s.getProfile)
}

func (s *Server) login(c *fiber.Ctx) error {
	return c.SendString("login")
}

func (s *Server) logout(c *fiber.Ctx) error {
	return c.SendString("logout")
}

func (s *Server) refreshToken(c *fiber.Ctx) error {
	return c.SendString("refresh token")
}

func (s *Server) getProfile(c *fiber.Ctx) error {
	return c.SendString("get profile")
}
