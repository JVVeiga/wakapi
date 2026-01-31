# Configuração

## Arquivo de Configuração

O Wakapi lê configuração de `config.yml` (padrão) ou do path especificado via `--config`. Todas as opções podem ser sobrepostas por variáveis de ambiente com prefixo `WAKAPI_`.

## Seções de Configuração

### Ambiente

| Campo | Tipo | Default | Env Var | Descrição |
|-------|------|---------|---------|-----------|
| `env` | string | production | WAKAPI_ENV | "dev"/"development" para modo dev |
| `quick_start` | bool | false | — | Pula tarefas iniciais (warmup de cache) |
| `skip_migrations` | bool | false | — | Não roda migrações (apenas dev) |
| `enable_pprof` | bool | false | — | Expõe endpoint de profiling |

### Server (`server`)

| Campo | Tipo | Default | Env Var |
|-------|------|---------|---------|
| `listen_ipv4` | string | 127.0.0.1 | — |
| `listen_ipv6` | string | ::1 | — |
| `listen_socket` | string | — | — |
| `port` | int | 3000 | — |
| `base_path` | string | / | — |
| `public_url` | string | http://localhost:3000 | — |
| `timeout_sec` | int | 30 | — |
| `tls_cert_path` | string | — | — |
| `tls_key_path` | string | — | — |

### App (`app`)

| Campo | Tipo | Default | Env Var |
|-------|------|---------|---------|
| `leaderboard_enabled` | bool | true | WAKAPI_LEADERBOARD_ENABLED |
| `leaderboard_scope` | string | 7_days | WAKAPI_LEADERBOARD_SCOPE |
| `leaderboard_generation_time` | string | 0 0 6 * * *,... | WAKAPI_LEADERBOARD_GENERATION_TIME |
| `aggregation_time` | string | 0 15 2 * * * | WAKAPI_AGGREGATION_TIME |
| `report_time_weekly` | string | 0 0 18 * * 5 | WAKAPI_REPORT_TIME_WEEKLY |
| `data_cleanup_time` | string | 0 0 6 * * 0 | WAKAPI_DATA_CLEANUP_TIME |
| `import_enabled` | bool | true | WAKAPI_IMPORT_ENABLED |
| `import_batch_size` | int | 50 | WAKAPI_IMPORT_BATCH_SIZE |
| `heartbeat_max_age` | string | 168h | WAKAPI_HEARTBEAT_MAX_AGE |
| `data_retention_months` | int | -1 | WAKAPI_DATA_RETENTION_MONTHS |
| `max_inactive_months` | int | -1 | WAKAPI_MAX_INACTIVE_MONTHS |
| `warm_caches` | bool | true | WAKAPI_WARM_CACHES |
| `custom_languages` | map | — | — |
| `default_language` | string | pt-BR | WAKAPI_DEFAULT_LANGUAGE | Idioma padrão da interface (pt-BR ou en) |
| `date_format` | string | Mon, 02 Jan 2006 | WAKAPI_DATE_FORMAT |
| `datetime_format` | string | Mon, 02 Jan 2006 15:04 | WAKAPI_DATETIME_FORMAT |

### Database (`db`)

| Campo | Tipo | Default | Env Var |
|-------|------|---------|---------|
| `dialect` | string | sqlite3 | WAKAPI_DB_TYPE |
| `host` | string | — | WAKAPI_DB_HOST |
| `port` | uint | — | WAKAPI_DB_PORT |
| `socket` | string | — | WAKAPI_DB_SOCKET |
| `user` | string | — | WAKAPI_DB_USER |
| `password` | string | — | WAKAPI_DB_PASSWORD |
| `name` | string | wakapi_db.db | WAKAPI_DB_NAME |
| `max_conn` | int | 10 | — |
| `ssl` | bool | false | — |
| `charset` | string | utf8mb4 | — |

**Dialetos suportados:** `sqlite3`, `postgres`, `mysql`, `mssql`

**Connection strings por dialeto:**
- **SQLite:** `file.db?busy_timeout=10000&journal_mode=wal`
- **MySQL:** `user:pass@tcp(host:port)/name?charset=utf8mb4&parseTime=true&loc=Local`
- **PostgreSQL:** `host=... port=... user=... dbname=... password=... sslmode=...`

### Security (`security`)

| Campo | Tipo | Default | Env Var |
|-------|------|---------|---------|
| `password_salt` | string | — | WAKAPI_PASSWORD_SALT |
| `insecure_cookies` | bool | false | WAKAPI_INSECURE_COOKIES |
| `cookie_max_age` | int | 172800 | WAKAPI_COOKIE_MAX_AGE |
| `allow_signup` | bool | true | WAKAPI_ALLOW_SIGNUP |
| `oidc_allow_signup` | bool | true | WAKAPI_OIDC_ALLOW_SIGNUP |
| `disable_local_auth` | bool | false | WAKAPI_DISABLE_LOCAL_AUTH |
| `signup_captcha` | bool | false | WAKAPI_SIGNUP_CAPTCHA |
| `invite_codes` | bool | true | WAKAPI_INVITE_CODES |
| `expose_metrics` | bool | false | WAKAPI_EXPOSE_METRICS |
| `enable_proxy` | bool | false | WAKAPI_ENABLE_PROXY |
| `disable_frontpage` | bool | true | WAKAPI_DISABLE_FRONTPAGE |
| `trusted_header_auth` | bool | false | WAKAPI_TRUSTED_HEADER_AUTH |
| `trusted_header_auth_key` | string | Remote-User | WAKAPI_TRUSTED_HEADER_AUTH_KEY |
| `signup_max_rate` | string | 5/1h | WAKAPI_SIGNUP_MAX_RATE |
| `login_max_rate` | string | 10/1m | WAKAPI_LOGIN_MAX_RATE |
| `password_reset_max_rate` | string | 5/1h | WAKAPI_PASSWORD_RESET_MAX_RATE |

**OIDC Providers:**
```yaml
security:
  oidc:
    - name: github
      display_name: GitHub
      client_id: xxx
      client_secret: xxx
      endpoint: https://github.com
```

### Mail (`mail`)

| Campo | Tipo | Default | Env Var |
|-------|------|---------|---------|
| `enabled` | bool | false | — |
| `provider` | string | smtp | — |
| `sender` | string | — | — |
| `smtp.host` | string | — | — |
| `smtp.port` | int | — | — |
| `smtp.username` | string | — | — |
| `smtp.password` | string | — | — |
| `smtp.tls` | bool | — | — |

### Sentry (`sentry`)

| Campo | Tipo | Default |
|-------|------|---------|
| `dsn` | string | — |
| `enable_tracing` | bool | true |
| `sample_rate` | float | 0.75 |
| `sample_rate_heartbeats` | float | 0.1 |

## Sistema de Migrações

### Como funciona

As migrações rodam em 3 fases:

1. **PreMigrations** — alterações que precisam rodar antes do AutoMigrate
2. **SchemaMigrations** — GORM AutoMigrate (cria/altera tabelas automaticamente)
3. **PostMigrations** — alterações que dependem do schema atualizado

### Padrão de uma migração

```go
func init() {
    const name = "20260111-nome_da_migracao"
    f := migrationFunc{
        name:       name,
        background: false, // true = pode rodar em goroutine
        f: func(db *gorm.DB, cfg *config.Config) error {
            if hasRun(name, db) {
                return nil // idempotente
            }
            // ... lógica da migração ...
            setHasRun(name, db)
            return nil
        },
    }
    registerPostMigration(f) // ou registerPreMigration(f)
}
```

### Idempotência

Cada migração verifica se já rodou consultando a tabela `key_string_values`. Isso permite re-executar migrações sem efeitos colaterais.

## Filas de Jobs

| Fila | Workers | Uso |
|------|---------|-----|
| QueueDefault | 1 | Dispatcher de crons |
| QueueProcessing | CPU/2 | Computações pesadas |
| QueueProcessing2 | CPU/2 | Computações pesadas (2) |
| QueueReports | 1 | Geração de relatórios |
| QueueMails | 1 | Envio de e-mails |
| QueueImports | 1 | Importação de dados |
| QueueHousekeeping | CPU/2 | Manutenção |

## Event Bus

Tópicos disponíveis:
- `user.*` → EventUserUpdate, EventUserDelete
- `heartbeat.*` → EventHeartbeatCreate
- `project_label.*`
- EventWakatimeFailure
- EventLanguageMappingsChanged
- EventApiKeyCreate, EventApiKeyDelete
