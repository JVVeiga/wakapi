# Serviços

Todas as interfaces estão definidas em `services/services.go`. A implementação segue o padrão de constructor injection.

## Grafo de Dependências

```
                    UserService
                   /    |    \
                  /     |     \
      HeartbeatService  |   ApiKeyService
         |     \        |       |
         |      \       |       |
    DurationService  SummaryService
         |              |    \
         |              |     \
   LanguageMappingService  AliasService
                        |
                  ProjectLabelService
```

## Serviços Principais

### UserService (`services/user.go`)
- **Dependências:** KeyValueService, MailService, ApiKeyService, UserRepository
- **Responsabilidades:** CRUD de usuários, autenticação, cache de usuários online
- **Cache:** 1h TTL, tracking de usuários online (30min TTL)
- **Eventos escutados:** `EventWakatimeFailure` (reseta API key), `EventHeartbeatCreate` (marca online)
- **Métodos-chave:** GetUserById, GetUserByKey, CreateOrGet, Update, Delete, ChangeUserId, ResetApiKey

### HeartbeatService (`services/heartbeat.go`)
- **Dependências:** HeartbeatRepository, LanguageMappingService
- **Responsabilidades:** Inserção/consulta de heartbeats, cache de contagem e projetos
- **Cache:** 24h TTL para contagens, sem expiração para entity sets
- **Eventos publicados:** `EventHeartbeatCreate` (a cada batch inserido)
- **Métodos-chave:** Insert, InsertBatch, Count, GetAllWithin, StreamAllWithin, GetUserProjectStats

### DurationService (`services/duration.go`)
- **Dependências:** DurationRepository, HeartbeatService, UserService, LanguageMappingService
- **Responsabilidades:** Converte heartbeats em durações (sessões de trabalho)
- **Algoritmo:** Agrupa heartbeats por gap máximo (timeout), calcula duração contínua
- **Métodos-chave:** Get (live ou cache), Regenerate, RegenerateAll

### SummaryService (`services/summary.go`)
- **Dependências:** SummaryRepository, HeartbeatService, DurationService, AliasService, ProjectLabelService
- **Responsabilidades:** Gera resumos agregados com breakdown por entidade
- **Cache:** 24h TTL por usuário
- **Métodos-chave:** Aliased (entry point principal), Retrieve (do banco), Summarize (do zero)

### AggregationService (`services/aggregation.go`)
- **Dependências:** UserService, SummaryService, HeartbeatService, DurationService
- **Responsabilidades:** Job em background que regenera summaries diariamente
- **Schedule:** Cron configurável (default: 02:15 AM)
- **Métodos-chave:** AggregateSummaries, AggregateDurations

## Serviços de Suporte

### AliasService (`services/alias.go`)
- Gerencia aliases de entidades (ex: "vim" → "Neovim")
- Cache: sync.Map por usuário, sem expiração
- Suporta wildcards em aliases

### ApiKeyService (`services/api_key.go`)
- Gerencia chaves de API (full access vs read-only)
- Cache: 24h, invalidado por eventos

### LanguageMappingService (`services/language_mapping.go`)
- Mapeia extensões de arquivo para linguagens
- Merge: server config + user config (user tem prioridade)
- Cache: 24h por usuário

### ProjectLabelService (`services/project_label.go`)
- Agrupa projetos por labels
- Cache: 24h, invalidado por eventos

### KeyValueService (`services/key_value.go`)
- Store key-value genérico
- Usado para estado de app, invite codes, contadores

## Serviços de Background

### ReportService (`services/report.go`)
- **Schedule:** Cron semanal (default: sexta 18h)
- Gera e envia relatórios por e-mail para usuários que optaram
- Throttle de 10s entre envios

### HousekeepingService (`services/housekeeping.go`)
- **Jobs:**
  - Limpeza de dados antigos (respeitando retenção)
  - Remoção de usuários inativos
  - Aquecimento de cache de project stats (a cada 12h)
  - Vacuum/Optimize do banco (mensal)

### LeaderboardService (`services/leaderboard.go`)
- **Schedule:** Múltiplos crons configuráveis
- Gera rankings individuais por tempo total, com agrupamento opcional por linguagem
- **Team leaderboard pré-computado:** após gerar rankings individuais, deriva leaderboard de times a partir dos `leaderboard_items` já existentes usando agregações SQL (SUM + GROUP BY), evitando O(N×M) queries on-the-fly
- Dados de times persistidos em tabela `team_leaderboard_items` (total por time, top 3 linguagens, contagem de membros)
- Cache: 6h TTL
- Reage a `EventUserUpdate` para auto-gerar/remover entries individuais e regenerar team leaderboard

### MiscService (`services/misc.go`)
- **Jobs:**
  - Contagem total de tempo de todos usuários (a cada 3h)
  - Notificação de subscriptions expirando (a cada 12h)

### ActivityService (`services/activity.go`)
- Gera gráficos SVG estilo GitHub contributions
- Cache: 6h TTL

## Sistema de E-mail (`services/mail/`)

```
MailService (orquestrador)
├── SendPasswordReset()
├── SendWakatimeFailureNotification()
├── SendImportNotification()
├── SendReport()
└── SendSubscriptionNotification()
    │
    ├── SMTPSendingService (produção)
    └── NoopSendingService (stub quando desabilitado)
```

## Jobs Agendados

| Serviço | Job | Frequência | Propósito |
|---------|-----|-----------|-----------|
| AggregationService | Summary Aggregation | Diário (cron) | Regenera summaries |
| ReportService | Report Generation | Semanal (cron) | Envia relatórios por e-mail |
| LeaderboardService | Leaderboard Individual | Múltiplos crons | Gera rankings individuais |
| LeaderboardService | Team Leaderboard | Múltiplos crons (após individual) | Gera rankings de times a partir dos dados individuais |
| HousekeepingService | Data Cleanup | Diário (cron) | Remove dados antigos |
| HousekeepingService | Inactive Users | Diário (cron) | Remove usuários inativos |
| HousekeepingService | Cache Warming | A cada 12h | Aquece cache de projetos |
| HousekeepingService | DB Optimization | Mensal (cron) | Vacuum/Optimize |
| MiscService | Total Time Count | A cada 3h | Conta tempo de todos usuários |
| MiscService | Subscription Notify | A cada 12h | Notifica subscriptions |

## Estratégia de Cache

| Serviço | Tipo | TTL | Invalidação |
|---------|------|-----|-------------|
| ActivityService | go-cache | 6h | Manual |
| ApiKeyService | go-cache | 24h | Event-driven |
| HeartbeatService | go-cache | 24h | Event/manual |
| LeaderboardService | go-cache | 6h | Manual |
| LanguageMappingService | go-cache | 24h | Event-driven |
| ProjectLabelService | go-cache | 24h | Event-driven |
| SummaryService | go-cache | 24h | Manual (por usuário) |
| UserService | go-cache | 1h | Manual (por usuário) |
| AliasService | sync.Map | Infinito | Manual (async) |

## Eventos (Pub/Sub)

| Evento | Publicador | Assinantes | Propósito |
|--------|-----------|------------|-----------|
| EventHeartbeatCreate | HeartbeatService | DurationService, UserService | Cache, tracking online |
| EventUserUpdate | UserService | LeaderboardService | Auto-gerar/remover leaderboard |
| EventWakatimeFailure | (externo) | UserService | Resetar API key |
| EventApiKeyCreate/Delete | ApiKeyService | ApiKeyService | Invalidar cache |
| TopicProjectLabel | ProjectLabelService | SummaryService | Invalidar cache |
