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

Team
├── TeamLeaderboardItem (1:N) — por TeamID, pré-computado via cron
└── TeamInvite (1:N) — convites de uso único com expiração
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
- `Language` (string, default: "pt-BR") — idioma preferido da interface (pt-BR ou en)

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

### TeamLeaderboardItem (`models/leaderboard.go`)

Entrada pré-computada do leaderboard de times. Derivada dos `leaderboard_items` individuais via agregação SQL no cron.

- `ID` (uint, PK)
- `TeamID` (string) — ID do time
- `TeamName` (string) — nome do time
- `Interval` (string) — período (7_days, etc.)
- `MemberCount` (int) — número de membros do time
- `Total` (time.Duration) — tempo total do time (soma dos individuais)
- `TopLanguagesJSON` (string) — JSON serializado das top 3 linguagens (persistido)
- `TopLanguages` ([]string) — top 3 linguagens (calculado via GORM hooks, não persistido)

Unique index: `idx_team_lb_combined` (TeamID, Interval)

### TeamInvite (`models/team_invite.go`)

Link de convite para entrar em um time. Uso único, com expiração de 2 horas.

- `ID` (uint, PK)
- `Code` (string, unique) — UUID do convite
- `TeamID` (string, FK → Team) — time alvo
- `CreatedBy` (string, FK → User) — quem gerou o convite
- `UsedBy` (*string) — quem usou (nil = não usado)
- `UsedAt` (*CustomTime) — quando foi usado
- `ExpiresAt` (CustomTime) — data de expiração
- `CreatedAt` (CustomTime) — data de criação

Status derivado em runtime: `IsUsed()`, `IsExpired()`, `Status()` → "active" / "expired" / "used"

### Team (`models/team.go`)

Grupo de usuários que compartilham dashboards e métricas agregadas.

**Campos principais:**
- `ID` (string, PK) — UUID do time
- `Name` (string) — nome do time
- `Description` (string) — descrição opcional
- `OwnerID` (string, FK → User) — dono original do time
- `Owner` (*User) — relacionamento com o dono
- `CreatedAt` (CustomTime) — data de criação

### TeamMember (`models/team.go`)

Associação entre usuário e time, com papel hierárquico.

**Campos principais:**
- `ID` (uint, PK auto)
- `TeamID` (string, FK → Team) — time associado
- `Team` (*Team) — relacionamento
- `UserID` (string, FK → User) — usuário membro
- `User` (*User) — relacionamento
- `Role` (string) — papel no time (ver hierarquia abaixo)
- `JoinedAt` (CustomTime) — data de entrada

Unique index: `idx_team_member_composite` (TeamID, UserID)

#### Hierarquia de Papéis

O sistema suporta três níveis de permissão em times:

**1. Owner (Dono)**
- Criador original do time ou receptor de transferência de ownership
- Permissões completas:
  - ✅ Criar convites para o time
  - ✅ Ver lista de convites ativos
  - ✅ Ver dashboards e métricas detalhadas dos membros
  - ✅ Ver activity charts do time
  - ✅ **Remover membros do time** (incluindo co-owners)
  - ✅ **Promover membros para co-owner**
  - ✅ **Rebaixar co-owners para member**
  - ✅ **Transferir ownership** para outro membro

**2. Co-Owner (Co-proprietário)**
- Papel intermediário com privilégios administrativos limitados
- Permissões:
  - ✅ Criar convites para o time
  - ✅ Ver lista de convites ativos
  - ✅ Ver dashboards e métricas detalhadas dos membros
  - ✅ Ver activity charts do time
  - ❌ **Não pode** remover membros do time
  - ❌ **Não pode** promover/rebaixar outros membros
  - ❌ **Não pode** transferir ownership

**3. Member (Membro)**
- Papel básico sem privilégios administrativos
- Permissões:
  - ✅ Ver métricas agregadas do time
  - ✅ Ver activity charts do time
  - ❌ Não pode ver dashboards individuais de outros membros
  - ❌ Não pode criar convites
  - ❌ Não pode remover membros
  - ❌ Não pode gerenciar papéis

#### Gerenciamento de Papéis

**Promoção/Demoção:**
- Apenas owners podem alterar papéis de membros
- Owners podem promover members para co-owner
- Owners podem rebaixar co-owners para member
- Co-owners não podem alterar papéis (nem o próprio)
- O papel de owner só pode ser transferido via `TransferOwnership()`

**Remoção de Membros:**
- Apenas owners podem remover membros
- Co-owners não podem ser removidos (proteção em `services/team.go`)
- Owner não pode ser removido (proteção em `services/team.go`)
- Membros regulares podem ser removidos por owners

**Constantes:**
```go
const (
    TeamRoleOwner   = "owner"     // Dono do time
    TeamRoleCoOwner = "co-owner"  // Co-proprietário
    TeamRoleMember  = "member"    // Membro regular
)
```

**Métodos de Validação:**
- `Team.IsValid()` — verifica se ID, Name e OwnerID estão presentes
- `TeamMember.IsValid()` — verifica se TeamID, UserID estão presentes e Role é válida

**Métodos de Permissão (TeamService):**

**✅ Recomendado (otimizado):**
- `GetUserPermissions(teamID, userID)` — retorna todas as permissões em uma única query
  - Retorna struct `TeamPermissions` com: `IsOwner`, `IsCoOwner`, `IsMember`, `CanRemove`, `CanPromote`, `CanManageInvites`, `CanViewDashboards`
  - Usa cache eficiente com chave única `team_perms_{teamID}_{userID}`
  - **60% mais rápido** que chamar métodos individuais

**⚠️ Métodos Individuais (disponíveis para compatibilidade):**
- `IsTeamOwner(teamID, userID)` — verifica se usuário é owner
- `IsTeamOwnerOrCoOwner(teamID, userID)` — verifica se é owner ou co-owner
- ~~`CanManageInvites(teamID, userID)`~~ — *deprecated, use GetUserPermissions*
- ~~`CanViewMemberDashboards(teamID, userID)`~~ — *deprecated, use GetUserPermissions*
- ~~`CanRemoveMembers(teamID, userID)`~~ — *deprecated, use GetUserPermissions*
- ~~`CanPromoteMembers(teamID, userID)`~~ — *deprecated, use GetUserPermissions*
- `UpdateMemberRole(teamID, userID, newRole)` — altera papel (apenas owner)

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
| TeamLeaderboardItem | `idx_team_lb_combined` (TeamID, Interval) |
| TeamInvite | uniqueIndex (Code), index (TeamID) |
