# Cursor Rules - Galaxy Pay

## 📋 Índice

- [Problema Identificado](#-problema-identificado)
- [Por Que Isso Era Um Problema](#-por-que-isso-era-um-problema)
- [Solução Implementada](#-solução-implementada)
- [Resultados Alcançados](#-resultados-alcançados)
- [Estrutura de Rules](#-estrutura-de-rules)
- [Como Funciona](#-como-funciona)
- [Métricas](#-métricas)
- [Lições Aprendidas](#-lições-aprendidas)

---

## 🚨 Problema Identificado

### Abordagem Inicial (Incorreta)

Inicialmente, criamos uma **única rule monolítica** para o framework Fiber:

```
.cursor/rules/fiber/RULE.md
└── 2,836 linhas (~20-40k tokens)
```

Esta rule continha **TODO** o conhecimento sobre Fiber em um único arquivo:
- Zero-allocation patterns
- Routing e Context API
- Middleware (30+ opções)
- Validation
- Error handling
- Hooks system
- Security
- Testing
- Architecture patterns

---

## 🔍 Por Que Isso Era Um Problema

### 1. Context Pollution

Modelos LLM têm um **limite de contexto** (20k-200k tokens). Com 30k tokens ocupados apenas pelas rules, sobrava pouco espaço para:
- Código real que você está editando
- Histórico da conversa
- Outros arquivos relevantes

### 2. Lost in the Middle Effect

Pesquisas mostram que LLMs têm **dificuldade em processar informações no meio de contextos grandes**. Com 3k linhas:
- ✅ Instruções no **início** são lembradas
- ❌ Instruções no **meio** são frequentemente ignoradas
- ✅ Instruções no **fim** (sua pergunta) são lembradas

**Resultado**: O modelo ignora partes importantes das regras.

### 3. Performance Issues

- **Latência**: Processar 30k tokens em cada mensagem deixa o chat mais lento
- **Custo**: Consome a cota de "fast requests" muito mais rápido
- **Processamento**: Mais tokens = mais tempo de processamento

### 4. Confusão Semântica

Em 3k linhas, é quase impossível evitar **instruções conflitantes**:
- Uma seção pede "código conciso"
- Outra pede "explicações detalhadas"
- Uma recomenda abordagem X
- Outra sugere alternativa Y

**Resultado**: O modelo fica confuso e pode alucinar.

### 5. Cenário Real

Imagine pedir ao Cursor para criar uma simples rota GET:

```go
// Você pede: "Crie uma rota GET /users"

// Cursor recebe:
// - 30k tokens de regras Fiber (incluindo middleware, validation, security, testing, etc.)
// - 2k tokens do seu código atual
// - 1k tokens da conversa

// Contexto útil para a tarefa: ~5%
// Contexto irrelevante: ~95%
```

É usar um **canhão para matar uma mosca** 🔫🦟

---

## ✅ Solução Implementada

### Modularização em Rules Focadas

Dividimos a rule monolítica em **9 rules especializadas**, cada uma com:

1. **Foco único** - Um tópico específico
2. **Tamanho otimizado** - Menos de 500 linhas (~3k tokens)
3. **Glob patterns** - Ativação automática em arquivos relevantes
4. **Descrição clara** - Para aplicação inteligente do Cursor Agent

### Nova Estrutura

```
.cursor/rules/
├── fiber-core/                    # 384 linhas
│   └── RULE.mdc                   # Zero-allocation, performance
├── fiber-routing/                 # 499 linhas
│   └── RULE.mdc                   # Routes, params, Context API
├── fiber-middleware/              # 434 linhas
│   └── RULE.mdc                   # Middleware patterns, ordering
├── fiber-validation/              # 233 linhas
│   └── RULE.mdc                   # go-playground/validator
├── fiber-error-handling/          # 278 linhas
│   └── RULE.mdc                   # Errors, panic recovery
├── fiber-hooks/                   # 315 linhas
│   └── RULE.mdc                   # Lifecycle hooks
├── fiber-security/                # 218 linhas
│   └── RULE.mdc                   # Auth, CORS, CSRF, Helmet
├── fiber-testing/                 # 242 linhas
│   └── RULE.mdc                   # Testing patterns
└── fiber-architecture/            # 349 linhas
    └── RULE.mdc                   # Galaxy Pay integration
```

**Total**: ~2,950 linhas distribuídas eficientemente
**Média**: ~328 linhas por rule
**Todas**: < 500 linhas ✅

### Exemplo de Frontmatter

Cada rule tem frontmatter otimizado:

```yaml
---
description: "Fiber routing patterns, dynamic parameters, route constraints, grouping, and Context API for request/response handling"
alwaysApply: false
globs: ["internal/handlers/**/*.go", "cmd/app/**/*.go"]
---
```

**O que isso faz**:
- `description`: Cursor Agent lê e decide se a rule é relevante
- `alwaysApply: false`: Não injeta sempre (só quando necessário)
- `globs`: Ativa automaticamente ao editar arquivos que correspondem ao padrão

---

## 🎯 Resultados Alcançados

### Performance

| Métrica | Antes | Depois | Melhoria |
|---------|-------|--------|----------|
| **Tokens por request** | ~30k | ~5-8k | **70-75% redução** |
| **Latência** | Alta | Baixa | **Respostas mais rápidas** |
| **Quota "fast requests"** | Consome rápido | Consome devagar | **Mais requisições disponíveis** |
| **Contexto para código** | ~30% | ~70-80% | **2.5x mais espaço** |

### Precisão

**Antes (Monolítico)**:
```
Tarefa: Criar rota GET
Contexto carregado:
✅ Routing (necessário)
❌ Middleware (desnecessário)
❌ Validation (desnecessário)
❌ Security (desnecessário)
❌ Testing (desnecessário)
❌ Hooks (desnecessário)
❌ Error handling (desnecessário)
```

**Depois (Modular)**:
```
Tarefa: Criar rota GET
Contexto carregado:
✅ fiber-routing (necessário - auto-ativado por glob)
✅ fiber-core (necessário - auto-ativado por glob)
✅ fiber-architecture (necessário - auto-ativado por glob)

Total: ~1,232 linhas (~8k tokens)
Relevância: ~100%
```

### Developer Experience

- ✅ **Menos confusão**: Rules focadas = instruções claras
- ✅ **Descoberta fácil**: Nomes descritivos mostram o que está disponível
- ✅ **Manutenção simples**: Atualizar validation não afeta routing
- ✅ **Escalabilidade**: Adicionar novas rules sem inflar as existentes

---

## 📚 Estrutura de Rules

### 1. fiber-core
**Propósito**: Conceitos fundamentais de performance
**Glob**: `**/*.go`
**Conteúdo**:
- Zero-allocation design philosophy
- Context value immutability
- `utils.CopyString()` patterns
- Custom JSON encoders
- Memory management

**Quando ativa**: Qualquer arquivo Go

---

### 2. fiber-routing
**Propósito**: Rotas e Context API
**Glob**: `internal/handlers/**/*.go`, `cmd/app/**/*.go`
**Conteúdo**:
- Route definition patterns
- Dynamic parameters (`:name`, `+`, `*`, `?`)
- Route constraints (`<int>`, `<guid>`, `<regex>`)
- Context API (Query, Params, Body, JSON, etc.)
- Route grouping

**Quando ativa**: Handlers e setup de rotas

---

### 3. fiber-middleware
**Propósito**: Middleware architecture
**Glob**: `internal/middleware/**/*.go`, `cmd/app/**/*.go`
**Conteúdo**:
- 30+ built-in middleware
- Critical ordering (Recover → Logger → Security → Performance)
- Custom middleware patterns
- Configuration examples

**Quando ativa**: Middleware files e app setup

---

### 4. fiber-validation
**Propósito**: Validation com go-playground/validator
**Glob**: `internal/handlers/**/*.go`, `internal/services/**/*.go`
**Conteúdo**:
- Validator wrapper setup
- Struct tag patterns
- Custom validation tags
- Error response structure

**Quando ativa**: Handlers e services (onde validação acontece)

---

### 5. fiber-error-handling
**Propósito**: Error handling patterns
**Glob**: `internal/handlers/**/*.go`, `cmd/app/**/*.go`
**Conteúdo**:
- Panic recovery
- `fiber.NewError()` usage
- Custom error handlers
- Structured error responses

**Quando ativa**: Handlers e app setup

---

### 6. fiber-hooks
**Propósito**: Lifecycle hooks
**Glob**: `cmd/app/**/*.go`
**Conteúdo**:
- 8 hooks disponíveis (OnRoute, OnListen, OnShutdown, etc.)
- Registration timing
- Common use cases

**Quando ativa**: App initialization (main.go)

---

### 7. fiber-security
**Propósito**: Security best practices
**Glob**: `internal/middleware/**/*.go`, `internal/handlers/**/*.go`
**Conteúdo**:
- BasicAuth, KeyAuth, JWT
- CSRF protection
- Helmet configuration
- CORS setup
- Secure cookies

**Quando ativa**: Security/auth files

---

### 8. fiber-testing
**Propósito**: Testing patterns
**Glob**: `**/*_test.go`
**Conteúdo**:
- Handler testing with `app.Test()`
- Middleware testing
- Integration tests
- Test helpers

**Quando ativa**: Test files apenas

---

### 9. fiber-architecture
**Propósito**: Integração com Galaxy Pay
**Glob**: `internal/handlers/**/*.go`, `internal/services/**/*.go`
**Conteúdo**:
- Handler → Service → Repository pattern
- Dependency injection
- Router organization
- Project structure

**Quando ativa**: Handlers e services (arquitetura em camadas)

---

## 🔧 Como Funciona

### Ativação Automática (Globs)

Quando você abre/edita um arquivo, o Cursor verifica os glob patterns:

```go
// Arquivo: internal/handlers/user_handler.go

// Rules ativadas automaticamente:
✅ fiber-routing (glob: "internal/handlers/**/*.go")
✅ fiber-core (glob: "**/*.go")
✅ fiber-architecture (glob: "internal/handlers/**/*.go")

// Total: ~1,232 linhas (~8k tokens)
// Relevância: Alta
```

```go
// Arquivo: internal/handlers/user_handler_test.go

// Rules ativadas automaticamente:
✅ fiber-testing (glob: "**/*_test.go")

// Total: ~242 linhas (~1.5k tokens)
// Relevância: Muito alta
```

### Aplicação Inteligente

O Cursor Agent também lê as **descriptions** e decide carregar rules adicionais se necessário:

```
Usuário: "Como faço para adicionar CORS no Fiber?"

Cursor Agent pensa:
- Descrição de fiber-security menciona "CORS"
- Carrega fiber-security mesmo que glob não corresponda

Resultado: Resposta precisa com a rule certa
```

---

## 📊 Métricas

### Comparação Direta

| Cenário | Antes (Monolítico) | Depois (Modular) | Melhoria |
|---------|-------------------|------------------|----------|
| **Criar rota GET simples** | 30k tokens | 8k tokens | **73% redução** |
| **Escrever teste** | 30k tokens | 1.5k tokens | **95% redução** |
| **Setup middleware** | 30k tokens | 7k tokens | **77% redução** |
| **Implementar validação** | 30k tokens | 6k tokens | **80% redução** |
| **Debugging** | 30k tokens | 5-10k tokens | **67-83% redução** |

### Economia de Tokens

**Média de redução**: ~75%
**Tokens economizados por request**: ~22k
**Mais contexto para código**: +150% (de ~30% → ~75%)

---

## 💡 Lições Aprendidas

### 1. Tamanho Importa

**Limite recomendado**: < 500 linhas por rule (~3k tokens)

- ✅ **< 300 linhas**: Excelente - muito focado
- ✅ **300-500 linhas**: Ótimo - bom equilíbrio
- ⚠️ **500-1000 linhas**: Aceitável - considerar dividir
- ❌ **> 1000 linhas**: Problemático - definitivamente dividir

### 2. Modularização > Monolítico

**Princípio**: "Carregue apenas o que você precisa, quando você precisa"

Melhor ter:
- 10 rules de 200 linhas cada (carrega 1-3 por vez)

Do que:
- 1 rule de 2000 linhas (carrega tudo sempre)

### 3. Globs São Poderosos

Use globs para **ativação contextual**:

```yaml
# Bom
globs: ["**/*_test.go"]  # Apenas em testes

# Ruim
globs: ["**/*.go"]  # Muito amplo
```

### 4. Descrições Importam

O Cursor Agent **lê as descrições** para decidir relevância:

```yaml
# ❌ Vago
description: "Fiber patterns"

# ✅ Específico
description: "Fiber routing patterns, dynamic parameters, route constraints, grouping, and Context API for request/response handling"
```

### 5. Evite Duplicação

Cada conceito deve estar em **uma única rule**:

- ❌ Context API em fiber-core E fiber-routing
- ✅ Context API apenas em fiber-routing

### 6. Mantenha Foco

Se uma seção não pertence ao tópico da rule, **mova para outra rule**:

```
fiber-routing:
  ✅ Route definition
  ✅ Parameters
  ✅ Context API
  ❌ Middleware (pertence a fiber-middleware)
  ❌ Validation (pertence a fiber-validation)
```

---

## 🎓 Referências

### Documentação Consultada

- [Cursor Rules Documentation](https://cursor.com/docs/context/rules)
- [Fiber Framework Documentation](https://docs.gofiber.io/)
- Community feedback: Reddit, Forums

### Artigos Relevantes

- **"Lost in the Middle"**: Como LLMs perdem foco em contextos grandes
- **Context Window Optimization**: Melhores práticas para uso eficiente de tokens
- **Prompt Engineering**: Estruturação de instruções para LLMs

---

## 📈 Próximos Passos

### Possíveis Melhorias

1. **Adicionar rules específicas**:
   - `fiber-websockets` - WebSocket patterns
   - `fiber-caching` - Cache strategies
   - `fiber-rate-limiting` - Rate limiting patterns

2. **Refinamento contínuo**:
   - Monitorar quais rules são mais usadas
   - Ajustar globs com base no uso real
   - Reduzir ainda mais rules muito grandes (> 400 linhas)

3. **Automação**:
   - Script para verificar tamanho das rules
   - CI/CD check para garantir limite de 500 linhas
   - Geração automática de métricas de uso

---

## 🤝 Contribuindo

Se você identificar:
- Rules que podem ser divididas
- Conteúdo duplicado entre rules
- Oportunidades de otimização

Sinta-se à vontade para sugerir melhorias!

---

## 📝 Conclusão

A modularização das Cursor rules foi um **case de sucesso** em otimização de contexto para LLMs:

- **Problema**: Rule monolítica de 3k linhas causando context pollution
- **Solução**: Modularização em 9 rules focadas com globs inteligentes
- **Resultado**: 70-80% redução de tokens, respostas mais rápidas, maior precisão

**Lição principal**: Em engenharia de prompts para LLMs, **menos é mais** quando bem organizado.

---

**Autor**: Gustavo Gomes
**Data**: Janeiro 2026
**Versão**: 1.0.0
