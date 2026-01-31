# Arquitetura do Wakapi

## Visão Geral

Wakapi é uma aplicação de tracking de tempo de desenvolvimento compatível com a API do WakaTime. Escrito em Go, usa uma arquitetura em camadas com Chi router, GORM ORM e suporte a múltiplos bancos de dados.

## Estrutura de Diretórios

```
wakapi/
├── main.go                      # Entrypoint da aplicação
├── config/                      # Sistema de configuração
│   ├── config.go                # Struct principal e Load()
│   ├── db.go                    # Dialectors e connection strings
│   ├── db_opts.go               # Hooks pós-inicialização do GORM
│   ├── eventbus.go              # Event bus (pub/sub)
│   ├── jobqueue.go              # Filas de jobs em background
│   ├── logging.go               # Inicialização do slog
│   ├── oidc.go                  # OpenID Connect providers
│   ├── session.go               # Cookie store para sessões
│   ├── sentry.go                # Integração Sentry
│   ├── templates.go             # Constantes de templates
│   ├── fs.go                    # Seleção de filesystem (embed vs local)
│   └── key_utils.go             # Geração de chaves criptográficas
├── models/                      # Entidades de domínio (GORM structs)
│   ├── user.go                  # Usuário
│   ├── heartbeat.go             # Evento de atividade
│   ├── duration.go              # Sessão de trabalho calculada
│   ├── summary.go               # Resumo agregado
│   ├── alias.go                 # Aliases de entidades
│   ├── api_key.go               # Chaves de API
│   ├── project_label.go         # Labels de projetos
│   ├── language_mapping.go      # Mapeamento extensão → linguagem
│   ├── leaderboard.go           # Item do leaderboard (usuários e times)
│   ├── team.go                  # Time e membros de time
│   ├── diagnostics.go           # Dados de diagnóstico
│   ├── shared.go                # KeyStringValue, CustomTime
│   ├── interval.go              # Intervalos de tempo
│   ├── filters.go               # Filtros de query
│   ├── mail.go                  # Modelo de e-mail
│   └── compat/                  # ViewModels para compatibilidade WakaTime
│       ├── wakatime/v1/         # Respostas no formato WakaTime
│       └── shields/v1/          # Respostas para shields.io
├── repositories/                # Camada de acesso a dados
│   ├── repositories.go          # Interfaces de todos os repositórios
│   ├── base.go                  # BaseRepository com helpers (batch, streaming)
│   ├── user.go                  # UserRepository
│   ├── heartbeat.go             # HeartbeatRepository
│   ├── duration.go              # DurationRepository
│   ├── summary.go               # SummaryRepository
│   ├── alias.go                 # AliasRepository
│   ├── api_key.go               # ApiKeyRepository
│   ├── key_value.go             # KeyValueRepository
│   ├── language_mapping.go      # LanguageMappingRepository
│   ├── project_label.go         # ProjectLabelRepository
│   ├── leaderboard.go           # LeaderboardRepository
│   ├── team.go                  # TeamRepository
│   ├── diagnostics.go           # DiagnosticsRepository
│   └── metrics.go               # MetricsRepository (Prometheus)
├── services/                    # Lógica de negócio
│   ├── services.go              # Interfaces de todos os serviços
│   ├── user.go                  # Gestão de usuários
│   ├── heartbeat.go             # Processamento de heartbeats
│   ├── duration.go              # Cálculo de durações
│   ├── summary.go               # Geração de resumos
│   ├── aggregation.go           # Agregação em background
│   ├── alias.go                 # Aliases
│   ├── api_key.go               # Chaves de API
│   ├── key_value.go             # Key-value store
│   ├── language_mapping.go      # Mapeamento de linguagens
│   ├── project_label.go         # Labels de projetos
│   ├── leaderboard.go           # Leaderboard
│   ├── team.go                  # Gestão de times e membros
│   ├── report.go                # Relatórios semanais por e-mail
│   ├── housekeeping.go          # Limpeza e manutenção
│   ├── misc.go                  # Tarefas diversas (contagem, notificações)
│   ├── activity.go              # Gráficos de atividade (SVG)
│   ├── diagnostics.go           # Diagnósticos
│   ├── mail/                    # Subsistema de e-mail
│   │   ├── mail.go              # Orquestrador + templates
│   │   ├── smtp.go              # Envio via SMTP
│   │   └── noop.go              # Stub (quando mail desabilitado)
│   └── imports/                 # Importação de dados
│       ├── importers.go         # Interface DataImporter
│       └── wakatime.go          # Importador WakaTime
├── routes/                      # Handlers HTTP
│   ├── handler.go               # Interface Handler
│   ├── routes.go                # Inicialização de templates
│   ├── home.go                  # Redirect para /login ou /summary
│   ├── login.go                 # Login/signup/OIDC/reset password
│   ├── summary.go               # Página de resumo
│   ├── settings.go              # Configurações do usuário
│   ├── projects.go              # Página de projetos
│   ├── leaderboard.go           # Leaderboard (requer autenticação)
│   ├── admin.go                 # Painel administrativo (requer IsAdmin)
│   ├── teams.go                 # Painel de times (métricas agregadas + dashboard individual de membros)
│   ├── subscription.go          # Assinatura Stripe
│   ├── imprint.go               # Página de impressum
│   ├── setup.go                 # Onboarding
│   ├── misc.go                  # Unsubscribe
│   ├── relay/relay.go           # Proxy relay para outras instâncias
│   ├── api/                     # Endpoints REST
│   │   ├── heartbeat.go         # POST /api/heartbeat(s)
│   │   ├── summary.go           # GET /api/summary
│   │   ├── health.go            # GET /api/health
│   │   ├── metrics.go           # GET /api/metrics (Prometheus)
│   │   ├── badge.go             # GET /api/badge (SVG)
│   │   ├── activity.go          # GET /api/activity/chart (SVG)
│   │   ├── avatar.go            # GET /api/avatar (SVG)
│   │   ├── diagnostics.go       # POST /api/plugins/errors
│   │   └── captcha.go           # GET /api/captcha
│   └── compat/                  # Compatibilidade WakaTime
│       ├── wakatime/v1/         # Endpoints WakaTime v1
│       └── shields/v1/          # Endpoints shields.io
├── middlewares/                  # Middleware HTTP
│   ├── authenticate.go          # Autenticação (cookie, API key, OIDC, header)
│   ├── principal.go             # Get/Set user no context
│   ├── logging.go               # Log de requests
│   ├── security.go              # Headers de segurança (CSP, X-Frame, etc.)
│   ├── filetype.go              # Filtro por tipo de arquivo
│   ├── shared_data.go           # Dados compartilhados no context
│   ├── sentry.go                # Integração Sentry
│   └── custom/wakatime.go       # Relay de heartbeats para WakaTime
├── migrations/                  # Migrações de banco
│   ├── migrations.go            # Runner (pre → schema → post)
│   ├── shared.go                # hasRun() / setHasRun()
│   └── 20YYMMDD_*.go            # Migrações individuais
├── helpers/                     # Funções utilitárias de domínio
├── utils/                       # Funções utilitárias gerais
├── mocks/                       # Mocks para testes
├── data/                        # Dados estáticos (colors.json)
├── static/                      # Assets embeddados (CSS, JS, imagens)
│   └── docs/                    # Swagger (swagger.yaml, swagger.json)
└── views/                       # Templates HTML (.tpl.html)
```

## Fluxo de Boot (main.go)

```
1. Parse CLI flags (--version, --config)
2. config.Load(configFlag, version)
   ├── Carrega YAML + variáveis de ambiente (WAKAPI_*)
   ├── Inicializa logger (slog)
   ├── Resolve dialeto do banco
   ├── Gera chaves criptográficas
   ├── Valida configurações
   └── Registra providers OIDC
3. Conecta ao banco (GORM)
4. migrations.Run(db, config)
   ├── RunPreMigrations()
   ├── RunSchemaMigrations() (AutoMigrate)
   └── RunPostMigrations()
5. Inicializa repositórios (12 instâncias)
6. Inicializa serviços (17+ instâncias, com DI)
7. Agenda jobs em background
   ├── aggregationService.Schedule()
   ├── reportService.Schedule()
   ├── housekeepingService.Schedule()
   ├── miscService.Schedule()
   └── leaderboardService.Schedule()
8. Cria handlers e registra rotas (Chi router)
9. listen() — inicia servidores IPv4/IPv6/Unix socket
```

## Diagrama de Camadas

```
┌─────────────────────────────────────────────┐
│              HTTP Request                    │
└──────────────────┬──────────────────────────┘
                   │
┌──────────────────▼──────────────────────────┐
│            MIDDLEWARE LAYER                   │
│  CleanPath → StripSlashes → Recoverer →      │
│  GetHead → SharedData → Logging → Sentry →   │
│  Security → Authenticate                     │
└──────────────────┬──────────────────────────┘
                   │
┌──────────────────▼──────────────────────────┐
│            HANDLER LAYER                     │
│  ┌────────────┐  ┌─────────────────────┐    │
│  │ routes/*   │  │ routes/api/*        │    │
│  │ (MVC/HTML) │  │ (REST/JSON)         │    │
│  └─────┬──────┘  └──────┬──────────────┘    │
└────────┼────────────────┼───────────────────┘
         │                │
┌────────▼────────────────▼───────────────────┐
│            SERVICE LAYER                     │
│  UserService, HeartbeatService, SummaryService│
│  DurationService, AliasService, etc.         │
└──────────────────┬──────────────────────────┘
                   │
┌──────────────────▼──────────────────────────┐
│          REPOSITORY LAYER                    │
│  UserRepo, HeartbeatRepo, SummaryRepo, etc.  │
└──────────────────┬──────────────────────────┘
                   │
┌──────────────────▼──────────────────────────┐
│          DATABASE (GORM)                     │
│  SQLite │ PostgreSQL │ MySQL │ SQL Server    │
└─────────────────────────────────────────────┘
```

## Ciclo de Vida de uma Request

```
1. Request HTTP chega
2. Middleware global: limpeza de path, logging, recovery
3. Router Chi despacha para rootRouter (/) ou apiRouter (/api)
4. Middleware de segurança (headers CSP, X-Frame-Options)
5. Middleware de autenticação:
   a. Tenta OIDC token
   b. Tenta cookie de sessão
   c. Tenta API key no header Authorization
   d. Tenta API key na query string
   e. Tenta trusted header (reverse proxy)
6. Handler processa a request
   - Extrai parâmetros
   - Chama serviços
   - Renderiza template ou retorna JSON
7. Response é enviada
```

## Tecnologias Principais

| Componente | Tecnologia |
|-----------|------------|
| Linguagem | Go |
| Router HTTP | Chi (go-chi/chi/v5) |
| ORM | GORM (gorm.io/gorm) |
| Bancos suportados | SQLite, PostgreSQL, MySQL, SQL Server |
| Templates | Go html/template |
| Sessões | gorilla/securecookie + gorilla/sessions |
| Jobs | artifex (filas com workers) |
| Eventos | leandro-lugaresi/hub (pub/sub) |
| Cache | patrickmn/go-cache |
| OIDC | coreos/go-oidc |
| E-mail | SMTP nativo |
| Monitoramento | Sentry |
| Métricas | Prometheus (formato texto) |
| API Docs | Swagger/OpenAPI (swaggo) |
| Assets | go:embed (estáticos embeddados) |
