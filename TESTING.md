# Workflow Engine - Testing Guide

Этот документ описывает стратегию тестирования Workflow Engine, включая unit tests, integration tests, load tests и CI/CD интеграцию.

## 📁 Структура тестов

```
platform-workflow/
├── engine/
│   ├── internal/
│   │   ├── core/bpmn/
│   │   │   ├── parser_test.go          # Unit tests для BPMN парсера
│   │   │   └── testdata/
│   │   │       └── simple.bpmn         # Тестовые BPMN файлы
│   │   └── ...
│   └── tests/
│       └── integration/
│           └── workflow_test.go        # Integration tests с testcontainers
├── tests/
│   └── load/
│       └── start_instance.js           # k6 load tests
├── .github/
│   └── workflows/
│       └── ci.yml                      # GitHub Actions CI
└── Makefile                            # Утилиты для запуска тестов
```

## 🧪 Unit Tests

### BPMN Parser Tests

Расположение: `engine/internal/core/bpmn/parser_test.go`

```bash
# Запуск unit tests
make test-unit

# Или напрямую
cd engine && go test -v ./internal/core/bpmn/...
```

**Тесты включают:**
- `TestParse_SimpleProcess` - парсинг валидного BPMN
- `TestParse_InvalidXML` - обработка невалидного XML
- `TestParse_MissingStartEvent` - валидация отсутствия стартового события
- `TestParse_ServiceTaskWithDelegate` - извлечение delegate expression
- `TestParse_GatewayWithConditions` - парсинг gateway с условиями

### Покрытие кода

```bash
make test-coverage
# Открыть coverage.html для просмотра отчета
```

## 🔗 Integration Tests

Integration tests используют [testcontainers-go](https://github.com/testcontainers/testcontainers-go) для запуска реальных PostgreSQL и NATS контейнеров.

### Запуск

```bash
make test-integration
```

### Что тестируется

- `TestFullWorkflow_HappyPath` - полный цикл: deploy → start → complete → verify
- `TestRegistryHeartbeat` - работа service registry
- `TestInstanceVariables` - работа с переменными процесса
- `TestWorkflow_RetryOnFailure` - ретраи при ошибках
- `TestWorkflow_Timeout` - таймауты

### Изоляция тестов

Каждый тест выполняется в изолированной транзакции с cleanup:
```go
func (s *WorkflowTestSuite) SetupTest() {
    s.cleanupDatabase()
}
```

## 📊 Load Tests (k6)

Load tests используют [k6](https://k6.io/) для нагрузочного тестирования.

### Сценарии

| Сценарий | Описание | VUs | Длительность |
|----------|----------|-----|--------------|
| smoke | Быстрая проверка работоспособности | 10 | 30s |
| load | Нагрузочное тестирование | 100-200 | 16m |
| stress | Стресс-тест до отказа | до 500 | 13m |
| spike | Резкий скачок нагрузки | 100→500 | 5m |

### Запуск

```bash
# Smoke tests
make load-test-smoke

# Load tests
make load-test-load

# Stress tests
make load-test-stress

# Spike tests
make load-test-spike
```

### Метрики

- **Latency**: p50 < 200ms, p95 < 500ms, p99 < 1000ms
- **Error Rate**: < 0.1%
- **Throughput**: requests per second
- **Instance Start Rate**: успешность создания инстансов

### Пороговые значения (Thresholds)

```javascript
export const options = {
  thresholds: {
    'http_req_duration': ['p(50) < 200', 'p(95) < 500', 'p(99) < 1000'],
    'http_req_failed': ['rate < 0.001'],
    'instance_start_rate': ['rate > 0.99'],
  },
};
```

### Отчеты

После запуска k6 генерирует:
- `reports/load-test-{timestamp}.json` - JSON отчет
- `reports/load-test-summary-{timestamp}.html` - HTML отчет

## 🔄 CI/CD Pipeline

GitHub Actions workflow: `.github/workflows/ci.yml`

### Stages

1. **Lint** - проверка кода golangci-lint
2. **Unit Tests** - unit tests с race detector
3. **Integration Tests** - integration tests с testcontainers
4. **Build** - сборка Docker образов
5. **Load Tests** - k6 нагрузочные тесты
6. **Security Scan** - Trivy security scan

### Запуск CI локально

```bash
make ci
```

## 🛠 Утилиты Makefile

### Тестирование

```bash
make test              # Все тесты
make test-unit         # Unit tests
make test-integration  # Integration tests
make test-bpmn         # BPMN parser tests
make test-coverage     # Coverage report
```

### Load Testing

```bash
make load-test-smoke   # Smoke tests
make load-test-load    # Load tests
make load-test-stress  # Stress tests
make load-test-spike   # Spike tests
```

### Разработка

```bash
make up                # Запуск всех сервисов
make down              # Остановка сервисов
make dev-logs          # Логи engine
make dev-shell         # Shell в engine контейнер
make db-migrate        # Миграции БД
make db-reset          # Сброс БД
```

## 🔧 Настройка окружения

### Требования

- Go 1.21+
- Docker & Docker Compose
- k6 (для load tests)
- Make

### Установка зависимостей

```bash
cd engine && go mod download
cd engine && go mod tidy
```

## 📋 Чек-лист перед коммитом

- [ ] Unit tests проходят: `make test-unit`
- [ ] Lint проходит: `make ci-lint`
- [ ] Код отформатирован: `make fmt`
- [ ] Новый функционал покрыт тестами
- [ ] Integration tests проходят (при наличии изменений в интеграциях)

## 🚨 Troubleshooting

### Integration tests падают с timeout

```bash
# Увеличить timeout
cd engine && go test -v ./tests/integration/... -timeout=30m
```

### Testcontainers не запускаются

```bash
# Проверить Docker
docker ps

# Очистить старые контейнеры
docker system prune -f
```

### k6 не установлен

```bash
# macOS
brew install k6

# Linux
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6
```

## 📚 Полезные ссылки

- [testcontainers-go](https://github.com/testcontainers/testcontainers-go)
- [k6 Documentation](https://k6.io/docs/)
- [Go Testing](https://golang.org/pkg/testing/)
- [testify](https://github.com/stretchr/testify)
