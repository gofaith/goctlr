package template

var (
	Imports = `import (
	"database/sql"
	"fmt"
	"strings"
	{{if .time}}"time"{{end}}

	"github.com/gofaith/go-zero/core/stores/cache"
	"github.com/gofaith/go-zero/core/stores/sqlc"
	"github.com/gofaith/go-zero/core/stores/sqlx"
	"github.com/gofaith/go-zero/core/stringx"
	"github.com/gofaith/goctl/model/sql/builderx"
)
`
	ImportsNoCache = `import (
	"database/sql"
	"strings"
	{{if .time}}"time"{{end}}

	"github.com/gofaith/go-zero/core/stores/sqlc"
	"github.com/gofaith/go-zero/core/stores/sqlx"
	"github.com/gofaith/go-zero/core/stringx"
	"github.com/gofaith/goctl/model/sql/builderx"
)
`
)
