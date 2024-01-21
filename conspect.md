Пакет logrus для логирования
go get github.com/sirupsen/logrus

Пакет zap для логирования
go get -u go.uber.org/zap

YAML
https://pkg.go.dev/gopkg.in/yaml.v3

TOML
https://github.com/pelletier/go-toml
import "github.com/pelletier/go-toml/v2"

JSON
Библиотека easyjson производит JSON-сериализацию структур, но, в отличие от реализации из стандартной библиотеки,
не использует рефлексию. Отсутствие рефлексии в JSON-сериализации (для всех форматов) значительно ускоряет операции
Marshal() и Unmarshal().
```
go get github.com/mailru/easyjson
go install github.com/mailru/easyjson/...@latest
```

MessagePack
MessagePack — это бинарный протокол сериализации, который требует предварительной генерации кода для работы с ним.
```
go install github.com/tinylib/msgp@latest
```

Protocol Buffers
Protocol Buffers (Protobuf) — это популярный в индустрии формат бинарного представления данных от компании Google.
Особенность протокола — наличие proto-файлов, которые описывают сериализуемые типы в своём формате (proto3).
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

PostgreSQL
```
go get -u github.com/lib/pq
go get -u github.com/jackc/pgx/v5
go get -u github.com/jackc/pgerrcode
go get -u github.com/golang-migrate/migrate/v4
```

JWT
```
go get -u github.com/golang-jwt/jwt/v4
```

UUID
```
go get -u github.com/google/uuid
```

Random
crypto/rand

Профилирование
```
go get -u github.com/gin-contrib/pprof
```

после того как ваше приложение запущено, используйте go tool pprof с флагом -alloc_space или -inuse_space
(в зависимости от того, что вы хотите измерить - используемую или выделенную память) и адресом HTTP-сервера
профилирования вашего приложения:
```
go run main.go
go tool pprof -http=:9090 http://localhost:8080/debug/pprof/heap
```

в веб-интерфейсе go tool pprof, в адресной строке вашего браузера, выполните команду сохранения профиля в файл
```
curl -sK -v http://localhost:8080/debug/pprof/heap > ./profiles/base.pprof
curl -sK -v http://localhost:8080/debug/pprof/heap > ./profiles/result.pprof
go tool pprof -top -diff_base=profiles/base.pprof profiles/result.pprof
```
p.s. (внимание) создает пустой файл, трубуется корректировка команды
```
go tool pprof -alloc_space -http=:9090 http://localhost:8080/debug/pprof/heap > profiles/base.pprof
go tool pprof -alloc_space -http=:9090 http://localhost:8080/debug/pprof/heap > profiles/result.pprof
```

Стилизация / Форматирование кода
```
gofmt -w main.go
goimports -local "github.com/myaccount/myproject" -w main.go
```

Документация
godoc
```
go install -v golang.org/x/tools/cmd/godoc@latest
```
Для локального отображения godoc-документации
```
godoc -http=:8080
```

Swagger
swag
```
$ go install github.com/swaggo/swag/cmd/swag@latest
```

после того как утилита swag установлена, Swagger-описание можно сгенерировать командой:
```
swag init --output ./swagger/ 
```

Шаблон example_test.go

multichecker
```
go get -u golang.org/x/tools
go get -u honnef.co/go/tools
go get -u golang.org/x/exp/typeparams
go get -u golang.org/x/mod
go get github.com/jackc/puddle/v2@v2.2.1
```

## Сборка с версионированием
`go build -ldflags "-X main.buildVersion=0.0.1 -X 'main.buildDate=$(date +'%Y/%m/%d %H:%M:%S')' -X main.buildCommit=xxx" cmd/shortner/main.go`