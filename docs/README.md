# Galaxy Pay - Documentação

Documentação completa do projeto Galaxy Pay, um serviço de pagamento desenvolvido em Go com Fiber.

## Índice

- [Arquitetura e Configuração](#arquitetura-e-configuração)
- [Sistema de Logging](#sistema-de-logging)
- [Request ID e Rastreabilidade](#request-id-e-rastreabilidade)
- [Idempotência](#idempotência)
- [Rate Limiting](#rate-limiting)
- [Configuração do Ambiente](#configuração-do-ambiente)
- [Docker Compose](#docker-compose)
- [Desenvolvimento](#desenvolvimento)

## Arquitetura e Configuração

### Estrutura do Projeto

```
galaxy-pay/
├── cmd/app/              # Ponto de entrada da aplicação
├── internal/
│   ├── config/           # Configurações centralizadas
│   ├── http/
│   │   └── middleware/    # Middlewares HTTP (Request ID, Logging, Rate Limit, Idempotency)
│   └── infrastructure/
│       └── cache/         # Cliente Redis e store de idempotência
├── pkg/
│   └── logger/           # Sistema de logging reutilizável
└── docs/                 # Documentação
```

### Configuração Centralizada

Todas as configurações são carregadas via variáveis de ambiente através do pacote `internal/config`. O arquivo `.env` é carregado automaticamente em desenvolvimento.

**Variáveis de Ambiente Principais:**

```env
# Aplicação
APP_NAME=Galaxy Pay
APP_VERSION=1.0.0
APP_ENV=development

# Servidor
HOST=0.0.0.0
PORT=8080
READ_TIMEOUT=15
WRITE_TIMEOUT=15
IDLE_TIMEOUT=60
BODY_LIMIT_MB=4

# Logging
LOG_LEVEL=info

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_POOL_SIZE=10
REDIS_MIN_IDLE_CONNS=5

# Rate Limiting
RATE_LIMIT_MAX=100
RATE_LIMIT_DURATION=60
```

## Sistema de Logging

### Visão Geral

O sistema de logging utiliza **Zap** (Uber) para logging estruturado de alta performance. O logger está localizado em `pkg/logger/` para facilitar reutilização em outros projetos.

### Características

- **Logging Estruturado**: JSON em produção, console colorido em desenvolvimento
- **Performance**: Zero-allocation onde possível
- **Níveis Configuráveis**: Debug, Info, Warn, Error
- **Integração com Request ID**: Todos os logs incluem request ID automaticamente

### Uso

```go
import "github.com/devgugga/galaxy-pay/pkg/logger"

// Inicialização (no main.go)
logger.Init(cfg.AppEnv, cfg.LogLevel)
defer logger.Sync()

// Uso básico
logger.Log.Info("Mensagem", logger.String("key", "value"))

// Com Request ID
logger.WithRequestID(requestID).Info("Request processada",
    logger.Method("POST"),
    logger.Path("/payments"),
    logger.Status(200),
)
```

### Formato de Logs

**Desenvolvimento (Console Colorido):**
```
[2026-01-05 10:30:45] 🟢 INFO  | HTTP Request | method=POST | path=/payments | status=200 | latency=1.2ms | req_id=abc-123
```

**Produção (JSON):**
```json
{
  "timestamp": "2026-01-05T10:30:45Z",
  "level": "info",
  "message": "HTTP Request",
  "method": "POST",
  "path": "/payments",
  "status": 200,
  "latency": "1.2ms",
  "request_id": "abc-123"
}
```

### Helpers Disponíveis

- `logger.RequestID(id)` - Campo para request ID
- `logger.Method(method)` - Método HTTP
- `logger.Path(path)` - Caminho da requisição
- `logger.Status(status)` - Status HTTP
- `logger.Latency(latency)` - Tempo de resposta
- `logger.IP(ip)` - IP do cliente
- `logger.Error(err)` - Erro
- `logger.String(key, value)` - String genérica
- `logger.Int(key, value)` - Inteiro
- `logger.Float64(key, value)` - Float64
- `logger.Bool(key, value)` - Boolean

## Request ID e Rastreabilidade

### Middleware de Request ID

Cada requisição recebe automaticamente um UUID v4 único que é:
- Gerado automaticamente se não fornecido
- Aceito via header `X-Request-ID` se fornecido pelo cliente
- Armazenado em `c.Locals("request_id")` para acesso em handlers
- Retornado no header de resposta `X-Request-ID`
- Incluído em todos os logs automaticamente

### Uso

```go
// O Request ID é adicionado automaticamente pelo middleware
// Acessar em handlers:
requestID := middleware.GetRequestID(c)

// Usar em logs:
logger.WithRequestID(requestID).Info("Processando pagamento")
```

### Ordem dos Middlewares

A ordem é crítica para garantir que o Request ID esteja disponível:

1. **Recover** - Captura panics
2. **Request ID** - Gera/extrai request ID
3. **Zap Logger** - Registra requisições (usa Request ID)
4. **Rate Limiting** - Limita requisições
5. **Idempotency** - Aplica idempotência (apenas em rotas específicas)

## Idempotência

### Conceito

Idempotência garante que múltiplas execuções da mesma requisição produzam o mesmo resultado, sem efeitos colaterais adicionais. **Essencial para APIs de pagamento** para evitar cobranças duplicadas.

### Implementação

O middleware de idempotência:
- Valida obrigatoriamente o header `Idempotency-Key` em POST/PUT/PATCH
- Armazena respostas em Redis com TTL de 24 horas
- Retorna resposta em cache se a mesma chave for usada novamente
- Processa normalmente se a chave não existir

### Fluxo

```
1. Cliente envia requisição com Idempotency-Key: <uuid>
2. Middleware verifica se existe resposta em cache
3. Se existe: retorna resposta armazenada (sem processar)
4. Se não existe: processa requisição e armazena resposta
5. Retorna resposta ao cliente
```

### Uso

```go
// Aplicar apenas em rotas de pagamento
payments := app.Group("/api/v1/payments")
payments.Use(middleware.Idempotency(idempotencyStore))
payments.Post("/", paymentHandler.Create)
```

### Requisitos

- **Header obrigatório**: `Idempotency-Key: <uuid>`
- **Formato**: UUID válido (mínimo 32 caracteres)
- **TTL**: 24 horas (configurável)
- **Armazenamento**: Redis

### Respostas de Erro

```json
// Sem Idempotency-Key
{
  "error": true,
  "message": "Idempotency-Key header is required for this operation"
}

// Formato inválido
{
  "error": true,
  "message": "Invalid Idempotency-Key format"
}
```

## Rate Limiting

### Implementação

Rate limiting é aplicado globalmente usando o middleware do Fiber, limitando requisições por IP.

### Configuração

```env
RATE_LIMIT_MAX=100        # Requisições por janela
RATE_LIMIT_DURATION=60    # Duração da janela em segundos
```

### Comportamento

- Limite aplicado por IP
- Headers informativos na resposta:
  - `X-RateLimit-Limit`: Limite máximo
  - `X-RateLimit-Remaining`: Requisições restantes
  - `X-RateLimit-Reset`: Timestamp de reset

### Resposta ao Exceder Limite

```json
{
  "error": true,
  "message": "Rate limit exceeded. Please try again later.",
  "status": 429
}
```

## Configuração do Ambiente

### Arquivo .env

Crie um arquivo `.env` na raiz do projeto baseado no `.env.example`:

```env
# Application Configuration
APP_NAME=Galaxy Pay
APP_VERSION=1.0.0
APP_ENV=development

# Server Configuration
HOST=0.0.0.0
PORT=8080

# Server Timeouts (in seconds)
READ_TIMEOUT=15
WRITE_TIMEOUT=15
IDLE_TIMEOUT=60

# Body Limit (in MB)
BODY_LIMIT_MB=4

# Logging Configuration
LOG_LEVEL=info

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_POOL_SIZE=10
REDIS_MIN_IDLE_CONNS=5

# Rate Limiting Configuration
RATE_LIMIT_MAX=100
RATE_LIMIT_DURATION=60
```

### Redis - Conexão Obrigatória

⚠️ **Importante**: Redis é **obrigatório** para a aplicação funcionar. A aplicação:

- Tenta conectar 5 vezes com delay exponencial
- **Não inicia** se não conseguir conectar
- Loga cada tentativa de conexão
- Encerra com erro fatal se todas as tentativas falharem

## Docker Compose

### Serviços

- **PostgreSQL 16** (Alpine) - Otimizado para performance
- **Redis 7.2** (Alpine) - Com AOF habilitado

Veja a [documentação completa do Docker Compose](docker-compose.md).

### Uso Rápido

```bash
# Iniciar serviços
docker-compose up -d

# Ver logs
docker-compose logs -f

# Parar
docker-compose down
```

## Desenvolvimento

### Executar Localmente

```bash
# Com Air (hot reload)
air

# Diretamente
go run ./cmd/app

# Build
go build -o tmp/main.exe ./cmd/app
```

### Estrutura de Middlewares

A ordem dos middlewares é crítica:

```go
// 1. Recover (primeiro - captura panics)
app.Use(recover.New(...))

// 2. Request ID (precisa estar cedo)
app.Use(middleware.RequestID())

// 3. Logger (usa Request ID)
app.Use(middleware.ZapLogger())

// 4. Rate Limiting
app.Use(middleware.RateLimit(cfg))

// 5. Idempotency (apenas em rotas específicas)
payments.Use(middleware.Idempotency(idempotencyStore))
```

### Boas Práticas

1. **Logging**:
   - Sempre use campos estruturados
   - Inclua Request ID em logs importantes
   - Não logue dados sensíveis (senhas, CVV, tokens)

2. **Idempotência**:
   - Apenas em operações que modificam estado
   - Use UUIDs válidos como chaves
   - Configure TTL apropriado

3. **Rate Limiting**:
   - Ajuste limites por tipo de endpoint
   - Monitore headers de resposta

4. **Redis**:
   - Sempre verifique se está conectado antes de usar
   - Use contextos com timeout
   - Trate erros de conexão adequadamente

## Segurança

### Medidas Implementadas

1. **Request ID**: Rastreabilidade completa de requisições
2. **Rate Limiting**: Proteção contra DDoS e abuso
3. **Idempotência**: Prevenção de operações duplicadas
4. **Logging Estruturado**: Auditoria completa
5. **Validação de Entrada**: Headers obrigatórios validados

### Recomendações para Produção

1. Configure HTTPS obrigatório
2. Use secrets do Docker para senhas
3. Configure firewall adequadamente
4. Monitore logs estruturados
5. Ajuste limites de rate limiting
6. Configure alertas para falhas de Redis
7. Implemente autenticação/autorização
8. Use validação rigorosa de dados

## Troubleshooting

### Redis não conecta

- Verifique se Redis está rodando: `docker-compose ps`
- Verifique logs: `docker-compose logs redis`
- Verifique variáveis de ambiente no `.env`
- A aplicação não inicia sem Redis (comportamento esperado)

### Logs não aparecem

- Verifique `LOG_LEVEL` no `.env`
- Em desenvolvimento, logs aparecem coloridos no console
- Em produção, logs são JSON estruturados

### Rate limit muito restritivo

- Ajuste `RATE_LIMIT_MAX` e `RATE_LIMIT_DURATION` no `.env`
- Reinicie a aplicação após alterar

### Idempotency não funciona

- Verifique se Redis está conectado
- Verifique se o header `Idempotency-Key` está sendo enviado
- Verifique logs para erros de cache

