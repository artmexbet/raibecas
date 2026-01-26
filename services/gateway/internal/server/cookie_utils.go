package server

import (
	"os"

	"github.com/gofiber/fiber/v2"
)

const (
	// Cookie names
	CookieRefreshToken = "refresh_token"
	CookieTokenID      = "token_id"
	CookieFingerprint  = "fingerprint"

	// Cookie settings
	RefreshTokenMaxAge = 30 * 24 * 60 * 60 // 30 days
	CookiePath         = "/"
	CookieDomain       = ""
)

// isProduction проверяет, запущено ли приложение в production режиме
func isProduction() bool {
	env := os.Getenv("ENVIRONMENT")
	// todo: возможно, стоит добавить проверку на другие значения
	return env == "production"
}

// getSameSite возвращает SameSite policy в зависимости от окружения
func getSameSite() string {
	if isProduction() {
		return "Strict" // Максимальная защита в production
	}
	return "Lax" // Более гибко для локальной разработки
}

// setSecureCookie sets a secure HttpOnly cookie
func setSecureCookie(c *fiber.Ctx, name, value string, maxAge int) {
	cookie := &fiber.Cookie{
		Name:     name,
		Value:    value,
		Path:     CookiePath,
		Domain:   CookieDomain,
		MaxAge:   maxAge,
		Secure:   isProduction(), // false в development для HTTP
		HTTPOnly: true,           // ALWAYS true для безопасности
		SameSite: getSameSite(),
	}
	c.Cookie(cookie)
}

// clearSecureCookie clears a secure cookie
func clearSecureCookie(c *fiber.Ctx, name string) {
	cookie := &fiber.Cookie{
		Name:     name,
		Value:    "",
		Path:     CookiePath,
		Domain:   CookieDomain,
		MaxAge:   -1,
		Secure:   isProduction(),
		HTTPOnly: true,
		SameSite: getSameSite(),
	}
	c.Cookie(cookie)
}

// getSecureCookie gets a secure cookie value
func getSecureCookie(c *fiber.Ctx, name string) string {
	return c.Cookies(name)
}
