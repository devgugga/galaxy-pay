# Docker Compose Setup

Este arquivo contém a configuração Docker Compose otimizada para PostgreSQL e Redis, atualizada para 2025/2026.

## Serviços Incluídos

### PostgreSQL 16 (Alpine)
- **Versão**: 16-alpine (mais recente e estável)
- **Porta**: 5432 (configurável via `POSTGRES_PORT`)
- **Otimizações**:
  - Configurações de performance ajustadas
  - Shared buffers: 256MB
  - Max connections: 200
  - Effective cache size: 1GB
  - WAL otimizado para alta performance
  - Parallel workers configurados

### Redis 7.2 (Alpine)
- **Versão**: 7.2-alpine (mais recente)
- **Porta**: 6379 (configurável via `REDIS_PORT`)
- **Otimizações**:
  - AOF (Append Only File) habilitado
  - Max memory: 512MB (configurável)
  - Policy: allkeys-lru
  - TCP keepalive configurado

## Uso Básico

### Iniciar serviços
```bash
docker-compose up -d
```

### Parar serviços
```bash
docker-compose down
```

### Parar e remover volumes (⚠️ apaga dados)
```bash
docker-compose down -v
```

### Ver logs
```bash
# Todos os serviços
docker-compose logs -f

# Apenas PostgreSQL
docker-compose logs -f postgres

# Apenas Redis
docker-compose logs -f redis
```

## Variáveis de Ambiente

Crie um arquivo `.env` na raiz do projeto com as seguintes variáveis:

```env
# PostgreSQL
POSTGRES_USER=galaxy_pay
POSTGRES_PASSWORD=your_secure_password
POSTGRES_DB=galaxy_pay
POSTGRES_PORT=5432

# Redis
REDIS_PORT=6379
```

## Configuração Local (Override)

Para personalizar configurações locais sem modificar o arquivo principal:

1. Copie o arquivo de exemplo:
```bash
cp docker-compose.override.yml.example docker-compose.override.yml
```

2. Edite `docker-compose.override.yml` conforme necessário

O arquivo `docker-compose.override.yml` é automaticamente carregado pelo Docker Compose e sobrescreve as configurações do `docker-compose.yml`.

## Health Checks

Ambos os serviços possuem health checks configurados:

- **PostgreSQL**: Verifica se está pronto para aceitar conexões
- **Redis**: Verifica se responde ao comando PING

Os health checks são executados a cada 10 segundos.

## Volumes

Os dados são persistidos em volumes nomeados:

- `galaxy-pay-postgres-data`: Dados do PostgreSQL
- `galaxy-pay-redis-data`: Dados do Redis (AOF)

## Network

Todos os serviços estão na mesma rede isolada: `galaxy-pay-network`

## Acessos

### PostgreSQL
- **Host**: localhost (ou `postgres` dentro da rede Docker)
- **Porta**: 5432 (ou a configurada em `POSTGRES_PORT`)
- **Usuário**: Configurado em `POSTGRES_USER`
- **Senha**: Configurada em `POSTGRES_PASSWORD`
- **Database**: Configurado em `POSTGRES_DB`

### Redis
- **Host**: localhost (ou `redis` dentro da rede Docker)
- **Porta**: 6379 (ou a configurada em `REDIS_PORT`)

## Integração com a Aplicação

Atualize seu arquivo `.env` da aplicação:

```env
# Database
DATABASE_URL=postgres://galaxy_pay:your_password@localhost:5432/galaxy_pay?sslmode=disable

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
```

## Comandos Úteis

### Backup do PostgreSQL
```bash
docker-compose exec postgres pg_dump -U galaxy_pay galaxy_pay > backup.sql
```

### Restore do PostgreSQL
```bash
docker-compose exec -T postgres psql -U galaxy_pay galaxy_pay < backup.sql
```

### Acessar PostgreSQL CLI
```bash
docker-compose exec postgres psql -U galaxy_pay -d galaxy_pay
```

### Acessar Redis CLI
```bash
docker-compose exec redis redis-cli
```

### Limpar dados do Redis
```bash
docker-compose exec redis redis-cli FLUSHALL
```

### Ver estatísticas do Redis
```bash
docker-compose exec redis redis-cli INFO
```

## Troubleshooting

### Serviço não inicia
```bash
# Ver logs detalhados
docker-compose logs [service-name]

# Verificar status
docker-compose ps

# Reiniciar serviço
docker-compose restart [service-name]
```

### Porta já em uso
Altere a porta no arquivo `.env` ou `docker-compose.override.yml`

### Problemas de permissão
```bash
# Remover volumes e recriar
docker-compose down -v
docker-compose up -d
```

## Performance

As configurações estão otimizadas para desenvolvimento. Para produção, ajuste:

- **PostgreSQL**: Aumente `shared_buffers` e `max_connections` conforme necessário
- **Redis**: Ajuste `maxmemory` baseado no uso esperado
- **Resource limits**: Ajuste `deploy.resources` conforme o hardware disponível

## Segurança

⚠️ **Importante para Produção**:

1. Altere todas as senhas padrão
2. Não exponha portas diretamente (use reverse proxy)
3. Configure SSL/TLS para PostgreSQL
4. Use secrets do Docker para senhas
5. Configure firewall adequadamente
6. Use redes Docker isoladas

