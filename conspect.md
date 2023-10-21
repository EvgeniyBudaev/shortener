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
go get github.com/mailru/easyjson
go install github.com/mailru/easyjson/...@latest

MessagePack
MessagePack — это бинарный протокол сериализации, который требует предварительной генерации кода для работы с ним.
go install github.com/tinylib/msgp@latest

Protocol Buffers
Protocol Buffers (Protobuf) — это популярный в индустрии формат бинарного представления данных от компании Google.
Особенность протокола — наличие proto-файлов, которые описывают сериализуемые типы в своём формате (proto3).
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

PostgreSQL
go get -u github.com/lib/pq
go get -u github.com/jackc/pgx/v5
go get -u github.com/jackc/pgerrcode
go get -u github.com/golang-migrate/migrate/v4

JWT
go get -u github.com/golang-jwt/jwt/v4

UUID
go get -u github.com/google/uuid

Random
crypto/rand