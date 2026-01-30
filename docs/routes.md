# Rotas e Endpoints

## Middleware Global (aplicado a todas as rotas)

1. **CleanPath** — normaliza paths
2. **StripSlashes** — remove trailing slashes
3. **Recoverer** — recovery de panics
4. **GetHead** — converte HEAD em GET
5. **SharedDataMiddleware** — dados compartilhados no context
6. **LoggingMiddleware** — log de requests (exclui /assets, /favicon, /api/health)
7. **SentryMiddleware** — error tracking (se configurado)

## Rotas MVC (Web UI)

Montadas em `rootRouter` com `SecurityMiddleware` (headers CSP, X-Frame-Options).

| Método | Path | Handler | Auth | Descrição |
|--------|------|---------|------|-----------|
| GET | `/` | HomeHandler.GetIndex | Opcional | Redirect para /summary (autenticado) ou /login |
| GET | `/login` | LoginHandler.GetIndex | Não | Formulário de login |
| POST | `/login` | LoginHandler.PostLogin | Não | Processar login (rate limited) |
| GET | `/signup` | LoginHandler.GetSignup | Não | Formulário de cadastro |
| POST | `/signup` | LoginHandler.PostSignup | Não | Processar cadastro (rate limited) |
| GET | `/set-password` | LoginHandler.GetSetPassword | Não | Formulário de nova senha |
| POST | `/set-password` | LoginHandler.PostSetPassword | Não | Processar nova senha |
| GET | `/reset-password` | LoginHandler.GetResetPassword | Não | Formulário de reset |
| POST | `/reset-password` | LoginHandler.PostResetPassword | Não | Processar reset (rate limited) |
| GET | `/oidc/{provider}/login` | LoginHandler.GetOidcLogin | Não | Inicia fluxo OIDC |
| GET | `/oidc/{provider}/callback` | LoginHandler.GetOidcCallback | Não | Callback OIDC |
| POST | `/logout` | LoginHandler.PostLogout | Sim | Logout |
| GET | `/summary` | SummaryHandler.GetIndex | Sim (redirect) | Página de resumo |
| GET | `/settings` | SettingsHandler.GetIndex | Sim (redirect) | Configurações |
| POST | `/settings` | SettingsHandler.PostIndex | Sim (redirect) | Salvar configurações |
| GET | `/projects` | ProjectsHandler.GetIndex | Sim (redirect) | Lista de projetos |
| GET | `/leaderboard` | LeaderboardHandler.GetIndex | Sim (redirect) | Leaderboard (requer autenticação) |
| GET | `/imprint` | ImprintHandler.GetImprint | Não | Impressum legal |
| GET | `/setup` | SetupHandler.GetIndex | Opcional | Onboarding |
| GET | `/unsubscribe` | MiscHandler.GetUnsubscribe | Não | Desinscrever de e-mails |

## API REST

Montadas em `apiRouter` em `/api`.

### Core API

| Método | Path | Handler | Auth | Descrição |
|--------|------|---------|------|-----------|
| GET | `/api` | ApiRootHandler.Get | Não | Root redirect |
| GET | `/api/health` | HealthApiHandler.Get | Não | Health check |
| GET | `/api/summary` | SummaryApiHandler.Get | Sim | Resumo JSON com filtros |
| POST | `/api/heartbeat` | HeartbeatApiHandler.Post | Sim (full) | Enviar heartbeat |
| POST | `/api/heartbeats` | HeartbeatApiHandler.Post | Sim (full) | Enviar heartbeats (bulk) |
| GET | `/api/metrics` | MetricsApiHandler.Get | Sim | Métricas Prometheus |
| POST | `/api/plugins/errors` | DiagnosticsApiHandler.Post | Não | Diagnósticos do CLI |
| GET | `/api/badge/{user}/*` | BadgeApiHandler.Get | Opcional | Badge SVG |
| GET | `/api/activity/chart/{user}` | ActivityApiHandler.Get | Opcional | Gráfico SVG |
| GET | `/api/avatar/{hash}.svg` | AvatarApiHandler.Get | Não | Avatar SVG |
| GET | `/api/captcha/{id}.png` | CaptchaHandler | Não | Imagem CAPTCHA |

### API WakaTime v1 (Compatibilidade)

Todos os endpoints existem com os prefixes:
- `/api/v1/...`
- `/api/compat/wakatime/v1/...`

| Método | Path | Handler | Auth | Descrição |
|--------|------|---------|------|-----------|
| GET | `.../users/{user}/stats/{range}` | StatsHandler.Get | Opcional | Estatísticas |
| GET | `.../users/{user}/stats` | StatsHandler.Get | Opcional | Estatísticas (sem range) |
| GET | `.../users/{user}/summaries` | SummariesHandler.Get | Sim | Resumos por período |
| GET | `.../users/{user}/statusbar/{range}` | StatusBarHandler.Get | Sim | Dados para status bar |
| GET | `.../users/{user}/all_time_since_today` | AllTimeHandler.Get | Sim | Tempo total |
| GET | `.../users/{user}/projects` | ProjectsHandler.Get | Sim | Lista de projetos |
| GET | `.../users/{user}/heartbeats` | HeartbeatHandler.Get | Sim | Heartbeats por data |
| GET | `.../users/{user}/user_agents` | UserAgentsHandler.Get | Sim | User agents |
| GET | `.../users/{user}` | UsersHandler.Get | Sim | Perfil do usuário |
| GET | `.../leaders` | LeadersHandler.Get | Sim | Leaderboard (requer autenticação) |

### Heartbeat Endpoints (múltiplos aliases)

O endpoint de heartbeat aceita vários paths para compatibilidade:
```
POST /api/heartbeat
POST /api/heartbeats
POST /api/users/{user}/heartbeats
POST /api/users/{user}/heartbeats.bulk
POST /api/v1/users/{user}/heartbeats
POST /api/v1/users/{user}/heartbeats.bulk
POST /api/compat/wakatime/v1/users/{user}/heartbeats
POST /api/compat/wakatime/v1/users/{user}/heartbeats.bulk
```

### Shields.io

| Método | Path | Auth | Descrição |
|--------|------|------|-----------|
| GET | `/api/compat/shields/v1/{user}/*` | Opcional | Badge dados para shields.io |

### Arquivos Estáticos

| Path | Descrição |
|------|-----------|
| `/assets/*` | CSS, JS, imagens (gzip em produção) |
| `/swagger-ui/*` | Documentação interativa da API |
| `/contribute.json` | Dados de contribuição |

## Fluxo de Autenticação

### Cadeia de autenticação (AuthenticateMiddleware)

```
1. tryHandleOidc()          → Token OIDC (se expirado, redireciona)
2. tryGetUserByCookie()     → Cookie de sessão (securecookie)
3. tryGetUserByApiKeyHeader() → Header Authorization: Basic <key>
4. tryGetUserByApiKeyQuery()  → Query param ?api_key=<key>
5. tryGetUserByTrustedHeader() → Header trusted (reverse proxy)
6. Se nenhum: 401 ou redirect para /login
```

### Métodos de Autenticação

| Método | Formato | Uso |
|--------|---------|-----|
| Cookie | `wakapi_auth` (securecookie) | Web UI (browser) |
| API Key Header | `Authorization: Basic <base64(key)>` | CLI/API |
| API Key Query | `?api_key=<key>` | Badges, integrações |
| OIDC | OAuth2 flow | SSO (GitHub, Google, etc.) |
| Trusted Header | Header customizável (default: Remote-User) | Reverse proxy auth |

### Tipos de API Key

- **Full Access:** Pode enviar heartbeats e ler todos os dados
- **Read Only:** Só pode ler dados (não pode enviar heartbeats)
