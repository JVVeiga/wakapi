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
| GET | `/leaderboard` | LeaderboardHandler.GetIndex | Sim (redirect) | Leaderboard (requer autenticação). `?tab=teams` para ranking de times |
| GET | `/imprint` | ImprintHandler.GetImprint | Não | Impressum legal |
| GET | `/setup` | SetupHandler.GetIndex | Opcional | Onboarding |
| GET | `/unsubscribe` | MiscHandler.GetUnsubscribe | Não | Desinscrever de e-mails |
| GET | `/lang` | LanguageHandler.GetSwitch | Não | Trocar idioma (seta cookie + atualiza User.Language se logado) |
| GET | `/teams` | TeamsHandler.GetIndex | Sim (redirect) | Lista de times do usuário |
| GET | `/teams/{id}` | TeamsHandler.GetTeamDetail | Sim (membro) | Detalhe do time com métricas agregadas |
| GET | `/teams/{id}/members/{userID}` | TeamsHandler.GetMemberSummary | Sim (owner/co-owner/admin) | Dashboard individual de um membro do time |
| POST | `/teams/{id}/members/remove` | TeamsHandler.PostRemoveMember | Sim (owner/admin) | Remover membro do time (apenas owner, co-owners não podem) |
| POST | `/teams/{id}/members/{userID}/role` | TeamsHandler.PostUpdateMemberRole | Sim (owner/admin) | Alterar papel de um membro (promover para co-owner ou rebaixar para member) |
| GET | `/teams/{id}/invites` | TeamsHandler.GetTeamInvites | Sim (owner/co-owner/admin) | Histórico de convites do time |
| POST | `/teams/{id}/invites` | TeamsHandler.PostGenerateInvite | Sim (owner/co-owner/admin) | Gerar link de convite |
| GET | `/teams/invite/{code}` | TeamsHandler.GetAcceptInvite | Sim (redirect) | Tela de aceitação de convite |
| POST | `/teams/invite/{code}` | TeamsHandler.PostAcceptInvite | Sim (redirect) | Aceitar convite e entrar no time |
| GET | `/admin` | AdminHandler.GetDashboard | Sim (admin) | Dashboard administrativo |
| GET | `/admin/users/{id}` | AdminHandler.GetUserDetail | Sim (admin) | Detalhes de um usuário |
| POST | `/admin/users/{id}` | AdminHandler.PostUserAction | Sim (admin) | Ações admin (promover/demover) |
| GET | `/admin/teams` | AdminHandler.GetTeams | Sim (admin) | Lista de times |
| POST | `/admin/teams` | AdminHandler.PostCreateTeam | Sim (admin) | Criar time |
| GET | `/admin/teams/{id}` | AdminHandler.GetTeamDetail | Sim (admin) | Detalhes de um time |
| POST | `/admin/teams/{id}` | AdminHandler.PostTeamAction | Sim (admin) | Ações no time (editar/deletar) |
| POST | `/admin/teams/{id}/members` | AdminHandler.PostTeamMemberAction | Sim (admin) | Adicionar/remover membro, transferir ownership |

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
| OIDC | OAuth2 flow | SSO (GitHub, Google, etc.). Se `email_verified == true` e já existe usuário com o mesmo email, vincula automaticamente a conta OIDC ao usuário existente |
| Trusted Header | Header customizável (default: Remote-User) | Reverse proxy auth |

### Tipos de API Key

- **Full Access:** Pode enviar heartbeats e ler todos os dados
- **Read Only:** Só pode ler dados (não pode enviar heartbeats)

### Acesso a Dados de Outros Usuários via API

Além do próprio usuário e admins, **owners e co-owners de times** podem consultar dados dos membros dos seus times via API. Isso se aplica a todos os endpoints GET da API WakaTime v1:

| Endpoint | Owner/Co-owner pode acessar? |
|----------|------------------------------|
| `.../users/{user}/summaries` | Sim |
| `.../users/{user}/stats/{range}` | Sim (sem restrição de range) |
| `.../users/{user}/statusbar/{range}` | Sim |
| `.../users/{user}/all_time_since_today` | Sim |
| `.../users/{user}/projects` | Sim |
| `.../users/{user}/heartbeats` | Sim |
| `.../users/{user}/user_agents` | Sim |
| `.../users/{user}` | Sim |
| POST heartbeats | Não (apenas o próprio usuário) |

**Exemplo de uso:**
```bash
# Owner consulta summaries de um membro do time
curl -s \
  "https://host/api/compat/wakatime/v1/users/MemberName/summaries?start=2026-03-13&end=2026-03-13" \
  -H "Authorization: Basic $(echo -n 'API_KEY_DO_OWNER' | base64)"
```

**Regras de autorização (ordem de prioridade):**
1. `{user}` = `current` → retorna dados do próprio usuário autenticado
2. `{user}` = ID do próprio usuário → acesso direto
3. Usuário autenticado é admin → acesso a qualquer usuário
4. Usuário autenticado é owner/co-owner de um time que contém `{user}` → acesso permitido
5. Caso contrário → 401 Unauthorized

## MCP Server (Model Context Protocol)

Servidor MCP para integração com IAs (Claude Desktop, etc.), permitindo que líderes de times analisem dados de coding dos seus liderados via conversação com IA.

### Configuração

```yaml
mcp:
  enabled: true       # desabilitado por default
  path: /mcp          # path no servidor HTTP existente
```

Variáveis de ambiente: `WAKAPI_MCP_ENABLED`, `WAKAPI_MCP_PATH`

### Endpoints

| Método | Path | Descrição |
|--------|------|-----------|
| GET | `/api/mcp/sse` | Conexão SSE do MCP |
| POST | `/api/mcp/message` | Mensagens MCP |

### Autenticação

Usa a mesma API key do WakaTime, via header `Authorization: Basic <base64(API_KEY)>`.

**Configuração no Claude Desktop** (`claude_desktop_config.json`):
```json
{
  "mcpServers": {
    "wakapi": {
      "url": "https://host/api/mcp/sse",
      "headers": {
        "Authorization": "Basic <base64(API_KEY_DO_LIDER)>"
      }
    }
  }
}
```

### Tools Disponíveis

| Tool | Descrição | Parâmetros Principais |
|------|-----------|----------------------|
| `list_teams` | Lista times que você lidera | — |
| `get_member_summary` | Summary detalhado de um membro | `team_id`, `user_id`, `interval`/`from`/`to`, `project`, `language` |
| `get_team_overview` | Visão agregada do time (ranking, top projetos/linguagens) | `team_id`, `interval`/`from`/`to` |
| `compare_members` | Comparação lado a lado de membros | `team_id`, `user_ids[]`, `interval`/`from`/`to` |
| `get_activity_patterns` | Distribuição horária e estatísticas de sessão | `team_id`, `user_id`, `interval`/`from`/`to` |
| `get_project_analysis` | Quem trabalha em cada projeto | `team_id`, `project`, `interval`/`from`/`to` |
| `get_trend_analysis` | Comparação entre períodos (tendências) | `team_id`, `user_id`, intervalos atual/anterior |

### Autorização

Apenas **owners e co-owners** de times podem usar as tools. Membros regulares não têm acesso aos dados de outros membros via MCP.
