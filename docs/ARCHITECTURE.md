# Arquitetura do Galaxy Pay

## Visão Geral

Galaxy Pay é um serviço de pagamento construído com Go e Fiber, seguindo uma arquitetura em camadas (layered architecture) com foco em segurança, performance e rastreabilidade.

## Camadas da Aplicação

```
┌─────────────────────────────────────┐
│      HTTP Layer (Fiber)             │
│  - Handlers                         │
│  - Middlewares                      │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│      Service Layer                  │
│  - Business Logic                   │
│  - Validation                       │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│      Repository Layer               │
│  - Data Access                      │
│  - Database Queries                 │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│      Infrastructure                 │
│  - Database (PostgreSQL)            │
│  - Cache (Redis)                    │
└─────────────────────────────────────┘
```

## Componentes Principais

### 1. HTTP Layer (`internal/http/`)

Responsável por:
- Receber requisições HTTP
- Validação de entrada
- Formatação de respostas
- Integração com middlewares

### 2. Middlewares (`internal/http/middleware/`)

**Request ID** (`request_id.go`)
- Gera UUID único por requisição
- Propaga em todos os logs
- Retorna no header de resposta

**Zap Logger** (`zap_logger.go`)
- Logging estruturado de requisições
- Integração com Request ID
- Níveis baseados em status HTTP

**Rate Limiting** (`rate_limit.go`)
- Limite de requisições por IP
- Headers informativos
- Configurável via variáveis de ambiente

**Idempotency** (`idempotency.go`)
- Validação de Idempotency-Key
- Cache de respostas em Redis
- Retorno de respostas em cache

### 3. Service Layer (`internal/service/`)

Contém a lógica de negócio:
- Processamento de pagamentos
- Validações de negócio
- Orquestração entre repositories

### 4. Repository Layer (`internal/repository/`)

Acesso a dados:
- Queries ao banco de dados
- Operações CRUD
- Transações

### 5. Infrastructure (`internal/infrastructure/`)

**Cache** (`cache/`)
- Cliente Redis com pool de conexões
- Store de idempotência
- Health checks

### 6. Logger (`pkg/logger/`)

Sistema de logging reutilizável:
- Inicialização baseada em ambiente
- Formatadores customizados
- Helpers para campos estruturados

## Fluxo de uma Requisição

```
1. Requisição HTTP chega
   ↓
2. Recover Middleware (captura panics)
   ↓
3. Request ID Middleware (gera/extrai ID)
   ↓
4. Zap Logger Middleware (registra requisição)
   ↓
5. Rate Limiting Middleware (verifica limites)
   ↓
6. Idempotency Middleware (se aplicável)
   ↓
7. Handler processa requisição
   ↓
8. Service executa lógica de negócio
   ↓
9. Repository acessa dados
   ↓
10. Resposta é formatada e retornada
```

## Dependências Externas

### PostgreSQL
- Banco de dados principal
- Armazenamento persistente
- Transações ACID

### Redis
- Cache de idempotência
- Armazenamento temporário de respostas
- **Obrigatório** para funcionamento

## Padrões de Design

### Dependency Injection
Serviços e handlers recebem dependências via construtores:

```go
type PaymentHandler struct {
    paymentService *service.PaymentService
}

func NewPaymentHandler(ps *service.PaymentService) *PaymentHandler {
    return &PaymentHandler{paymentService: ps}
}
```

### Error Handling
- Erros são retornados e tratados pelo error handler global
- Logs estruturados incluem contexto do erro
- Respostas de erro são consistentes

### Configuration
- Configuração centralizada em `internal/config`
- Carregamento via variáveis de ambiente
- Valores padrão para desenvolvimento

## Segurança

### Camadas de Segurança

1. **Rate Limiting**: Proteção contra DDoS
2. **Idempotência**: Prevenção de duplicação
3. **Request ID**: Rastreabilidade completa
4. **Logging**: Auditoria de todas as operações
5. **Validação**: Headers e dados validados

### Boas Práticas

- Não logar dados sensíveis
- Validar todas as entradas
- Usar HTTPS em produção
- Implementar autenticação/autorização
- Monitorar logs estruturados

## Performance

### Otimizações

- **Zero-allocation logging**: Zap otimizado
- **Connection pooling**: Redis e PostgreSQL
- **Custom JSON encoder**: goccy/go-json
- **Efficient middleware**: Ordem otimizada
- **Resource limits**: Docker Compose configurado

### Métricas

- Request ID em todos os logs para rastreabilidade
- Latency registrado em cada requisição
- Status codes logados para monitoramento

## Escalabilidade

### Horizontal Scaling

- Stateless application (pode escalar horizontalmente)
- Redis compartilhado para idempotência
- PostgreSQL com connection pooling

### Vertical Scaling

- Resource limits configuráveis
- Connection pools ajustáveis
- Rate limits por instância

## Monitoramento

### Logs Estruturados

Todos os logs incluem:
- Timestamp
- Level
- Message
- Request ID
- Context (method, path, status, latency, IP)

### Health Checks

- `/health` - Health check básico
- `/ready` - Readiness check (verifica dependências)

## Próximos Passos

1. Implementar autenticação JWT
2. Adicionar métricas (Prometheus)
3. Implementar tracing distribuído
4. Adicionar testes automatizados
5. Configurar CI/CD

