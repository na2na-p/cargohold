package auth

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/na2na-p/cargohold/internal/handler/middleware"
)

func SessionDisplayHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		sessionID := c.QueryParam("session_id")
		if sessionID == "" {
			return middleware.NewAppError(
				http.StatusBadRequest,
				"session_idパラメータが指定されていません",
				errors.New("session_id parameter is empty"),
			)
		}

		host := c.QueryParam("host")
		if host == "" {
			return middleware.NewAppError(
				http.StatusBadRequest,
				"hostパラメータが指定されていません",
				errors.New("host parameter is empty"),
			)
		}

		html := generateSessionDisplayHTML(sessionID, host)
		return c.HTML(http.StatusOK, html)
	}
}

func generateSessionDisplayHTML(sessionID, host string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="ja">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>認証成功 - Cargohold</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            max-width: 800px;
            margin: 50px auto;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            background: white;
            padding: 30px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        h1 {
            color: #28a745;
            margin-bottom: 20px;
        }
        pre {
            background: #1e1e1e;
            color: #d4d4d4;
            padding: 20px;
            border-radius: 4px;
            overflow-x: auto;
            font-size: 14px;
        }
        .note {
            color: #666;
            font-size: 14px;
            margin-top: 20px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>認証成功！</h1>
        <p>Git LFS を使用するには、以下のコマンドを実行してください：</p>
        <pre>git credential approve &lt;&lt;EOF
protocol=https
host=%s
username=x-session
password=%s
EOF</pre>
        <p class="note">セッションの有効期限: 24時間</p>
    </div>
</body>
</html>`, host, sessionID)
}
