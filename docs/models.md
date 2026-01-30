# Modelos de Dados

## Diagrama de Relacionamentos

```
User (Entidade Principal)
├── Heartbeat (1:N) — CASCADE DELETE
├── Duration (1:N) — por UserID
├── Summary (1:N) — CASCADE DELETE
│   └── SummaryItem (1:N) — CASCADE DELETE
├── Alias (1:N) — CASCADE DELETE
├── ApiKey (1:N) — CASCADE DELETE
├── LanguageMapping (1:N) — CASCADE DELETE
├── ProjectLabel (1:N) — CASCADE DELETE
├── LeaderboardItem (1:N) — CASCADE DELETE
└── Diagnostics (sem FK direta)
```

## Entidades

### User (`models/user.go`)

Entidade principal do sistema. Armazena dados de autenticação, perfil e preferências.

**Campos principais:**
- `ID` (string, PK) — username
- `Email` (unique, nullable)
- `Password` (hash bcrypt)
- `ApiKey` (string) — chave padrão
- `Location` (string) — timezone
- `WakatimeApiKey` / `WakatimeApiUrl` — integração WakaTime
- `ShareDataMaxDays` — dias de dados públicos (0 = privado)
- `ShareEditors`, `ShareLanguages`, etc. — flags de compartilhamento
- `ReportsWeekly` (bool) — relatório semanal por e-mail
- `PublicLeaderboard` (bool) — participar do leaderboard
- `AuthType` (string, default: "local") — tipo de auth (local/oidc)
- `HeartbeatsTimeoutSec` (int, default: 600) — timeout entre heartbeats

### Heartbeat (`models/heartbeat.go`)

Evento individual de atividade do editor de código.

**Campos principais:**
- `ID` (uint64, PK auto)
- `UserID` (FK → User)
- `Entity` (string) — arquivo sendo editado
- `Type` (string) — tipo de entidade (file, domain, app)
- `Category` (string) — coding, browsing, debugging, etc.
- `Project`, `Branch`, `Language` — contexto do trabalho
- `Editor`, `OperatingSystem`, `Machine` — ambiente
- `IsWrite` (bool) — se é uma operação de escrita
- `Time` (CustomTime) — timestamp do evento
- `Hash` (string, unique) — deduplicação
- `UserAgent` (string) — client info

**Índices:** time, time+user, user+project, project, branch, language, editor, os, machine

### Duration (`models/duration.go`)

Sessão de trabalho calculada a partir de heartbeats.

**Campos principais:**
- `ID` (int64, PK auto)
- `UserID` (string)
- `Time` (CustomTime) — início da sessão
- `Duration` (time.Duration) — duração calculada
- `Project`, `Language`, `Editor`, `OperatingSystem`, `Machine`, `Branch`, `Category`
- `NumHeartbeats` (int) — heartbeats nessa sessão
- `Timeout` (time.Duration, default: 10min) — gap máximo entre heartbeats

### Summary (`models/summary.go`)

Resumo agregado de atividade por período.

**Campos principais:**
- `ID` (uint, PK)
- `UserID` (FK → User)
- `FromTime`, `ToTime` (CustomTime) — período coberto
- `Projects`, `Languages`, `Editors`, `OperatingSystems`, `Machines` (SummaryItems)
- `Labels`, `Branches`, `Entities`, `Categories` (calculados em runtime)
- `NumHeartbeats` (int)

**SummaryItem:**
- `Type` (uint8) — 0=Project, 1=Language, 2=Editor, 3=OS, 4=Machine, 5=Label, 6=Branch, 7=Entity, 8=Category
- `Key` (string) — nome da entidade
- `Total` (time.Duration) — tempo total

### Alias (`models/alias.go`)

Mapeamento de nomes alternativos para entidades.

- `ID` (uint, PK)
- `UserID` (FK → User)
- `Type` (uint8) — tipo de entidade (project, language, etc.)
- `Key` (string) — nome original
- `Value` (string) — nome canônico

### ApiKey (`models/api_key.go`)

Chaves de autenticação para API.

- `ApiKey` (string, PK)
- `UserID` (FK → User)
- `ReadOnly` (bool) — se é somente leitura
- `Label` (string) — descrição

### LanguageMapping (`models/language_mapping.go`)

Mapeamento customizado de extensão de arquivo para linguagem.

- `ID` (uint, PK)
- `UserID` (FK → User)
- `Extension` (string) — ex: ".vue"
- `Language` (string) — ex: "Vue"

### ProjectLabel (`models/project_label.go`)

Categorização de projetos com labels.

- `ID` (uint, PK)
- `UserID` (FK → User)
- `ProjectKey` (string) — nome do projeto
- `Label` (string) — label/categoria

### LeaderboardItem (`models/leaderboard.go`)

Entrada no leaderboard público.

- `ID` (uint, PK)
- `UserID` (FK → User)
- `Interval` (string) — período (7_days, etc.)
- `By` (*uint8) — agrupamento opcional (por linguagem, editor, etc.)
- `Total` (time.Duration) — tempo total
- `Key` (*string) — chave de agrupamento

### KeyStringValue (`models/shared.go`)

Store key-value genérico para estado da aplicação.

- `Key` (string, PK)
- `Value` (string)

Usado para: estado de migrações, invite codes, dados de importação, contadores, etc.

### CustomTime (`models/shared.go`)

Wrapper sobre `time.Time` que suporta unmarshalling de timestamps Python (formato `<sec>.<nsec>`). Implementa serialização JSON, scanning de banco, e conversão de tipos para cada dialeto SQL.

## Índices de Performance

| Modelo | Índices |
|--------|---------|
| User | `idx_user_email` |
| Heartbeat | `idx_time`, `idx_time_user`, `idx_user_project`, `idx_project`, `idx_branch`, `idx_language`, `idx_editor`, `idx_operating_system`, `idx_machine` |
| Duration | `idx_time_duration`, `idx_time_duration_user` |
| Summary | `idx_time_summary_user` (UserID, FromTime, ToTime) |
| SummaryItem | `idx_type` |
| Alias | `idx_alias_user`, `idx_alias_type_key` |
| ApiKey | `idx_api_key_user` |
| LanguageMapping | `idx_language_mapping_user`, unique (UserID, Extension) |
| ProjectLabel | `idx_project_label_user` |
| LeaderboardItem | `idx_leaderboard_user`, `idx_leaderboard_combined` |
