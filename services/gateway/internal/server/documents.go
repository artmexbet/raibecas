package server

import "github.com/gofiber/fiber/v2"

func (s *Server) setupDocumentRoutes() {
	documents := s.router.Group("/documents")
	documents.Get("/", s.listDocuments)
	documents.Post("/", s.createDocument)
	documents.Get("/:id", s.getDocument)
	documents.Put("/:id", s.updateDocument)
	documents.Delete("/:id", s.deleteDocument)
}

func (s *Server) listDocuments(c *fiber.Ctx) error {
	return c.SendString("list documents")
}

func (s *Server) createDocument(c *fiber.Ctx) error {
	return c.SendString("create document")
}

func (s *Server) getDocument(c *fiber.Ctx) error {
	return c.SendString("get document")
}

func (s *Server) updateDocument(c *fiber.Ctx) error {
	return c.SendString("update document")
}

func (s *Server) deleteDocument(c *fiber.Ctx) error {
	return c.SendString("delete document")
}
