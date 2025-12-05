package ingestion

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"github.com/artmexbet/raibecas/services/index/internal/domain"
)

type iPipeline interface {
	Index(ctx context.Context, doc domain.Document) error
}

type iStorage interface {
	Save(ctx context.Context, documentID string, reader io.Reader) (string, error)
}

type HTTPIngestor struct {
	app     *fiber.App
	pipe    iPipeline
	storage iStorage
}

func NewHTTPIngestor(pipe iPipeline, storage iStorage) *HTTPIngestor {
	app := fiber.New(fiber.Config{
		BodyLimit: 100 * 1024 * 1024, // 100MB max file size
	})

	app.Use(requestid.New())
	app.Use(logger.New())
	app.Use(recover.New())

	ingestor := &HTTPIngestor{app: app, pipe: pipe, storage: storage}

	app.Post("/api/v1/index", ingestor.indexFile())
	app.Post("/api/v1/index/json", ingestor.indexJSON())

	return ingestor
}

// indexFile принимает файл через multipart/form-data
func (i *HTTPIngestor) indexFile() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Получаем файл из формы
		file, err := c.FormFile("file")
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "file is required")
		}

		// Получаем метаданные
		documentID := c.FormValue("id")
		if documentID == "" {
			return fiber.NewError(fiber.StatusBadRequest, "id is required")
		}

		title := c.FormValue("title")
		sourceURI := c.FormValue("source_uri")

		// Парсим metadata из JSON строки если есть
		metadata := make(map[string]string)
		metadata["original_filename"] = file.Filename
		metadata["size"] = fmt.Sprintf("%d", file.Size)

		// Открываем файл
		fileReader, err := file.Open()
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to open file")
		}
		defer func() {
			_ = fileReader.Close()
		}()

		// Сохраняем файл в storage
		filePath, err := i.storage.Save(c.UserContext(), documentID, fileReader)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to save file: "+err.Error())
		}

		// Создаем документ с путем к файлу
		doc := domain.Document{
			ID:        documentID,
			Title:     title,
			FilePath:  filePath,
			SourceURI: sourceURI,
			Metadata:  metadata,
		}

		// Индексируем
		if err := i.pipe.Index(c.UserContext(), doc); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}

		return c.JSON(fiber.Map{
			"status":    "accepted",
			"id":        documentID,
			"file_path": filePath,
		})
	}
}

// indexJSON - legacy endpoint для обратной совместимости (deprecated)
func (i *HTTPIngestor) indexJSON() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req IndexRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}

		if req.ID == "" || req.Content == "" {
			return fiber.NewError(fiber.StatusBadRequest, "id and content are required")
		}

		doc := domain.Document{
			ID:        req.ID,
			Title:     req.Title,
			Content:   req.Content,
			SourceURI: req.SourceURI,
			Metadata:  req.Metadata,
		}

		if err := i.pipe.Index(c.UserContext(), doc); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}

		return c.SendStatus(fiber.StatusAccepted)
	}
}

func (i *HTTPIngestor) Start(addr string) error {
	return i.app.Listen(addr)
}

func (i *HTTPIngestor) Shutdown() error {
	return i.app.Shutdown()
}

// Test возвращает fiber.App для тестирования
func (i *HTTPIngestor) Test(req *http.Request, msTimeout ...int) (*http.Response, error) {
	return i.app.Test(req, msTimeout...)
}
