# InfraForge

A platform engineering tool for managing infrastructure environments with workflow orchestration, approval gates, drift detection, and audit logging.

Built with Go, Temporal, PostgreSQL, and Next.js.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                          Frontend (Next.js)                          │
│                         localhost:3000                               │
└──────────────────────────────┬──────────────────────────────────────┘
                               │ /api/v1/* (proxy)
┌──────────────────────────────▼──────────────────────────────────────┐
│                        API Server (Go)                               │
│                       localhost:8081                                  │
│  ┌──────────┐ ┌───────────┐ ┌──────────┐ ┌────────────────────┐    │
│  │  Teams   │ │Environments│ │Workflows │ │ Drift Simulator    │    │
│  │  CRUD    │ │   CRUD     │ │List/Retry│ │ (30s background)   │    │
│  └──────────┘ └───────────┘ └──────────┘ └────────────────────┘    │
│  ┌──────────┐ ┌───────────┐ ┌──────────┐ ┌────────────────────┐    │
│  │Approvals │ │   Drift   │ │  Audit   │ │    Dashboard       │    │
│  │Approve/  │ │Report/    │ │  Log     │ │    Stats           │    │
│  │Reject    │ │Resolve    │ │          │ │                    │    │
│  └──────────┘ └───────────┘ └──────────┘ └────────────────────┘    │
└───────────┬─────────────────────────────────────┬───────────────────┘
            │                                     │
            ▼                                     ▼
┌───────────────────────┐           ┌─────────────────────────────────┐
│    PostgreSQL 16      │           │      Temporal Server 1.24       │
│   localhost:5435      │           │       localhost:7233             │
│                       │           │                                 │
│  teams                │           │  Task Queue: infraforge-worker  │
│  environments         │           │                                 │
│  workflows            │           │  ┌─────────────────────────┐    │
│  workflow_steps       │           │  │   Temporal Worker (Go)  │    │
│  approvals            │           │  │                         │    │
│  audit_log            │           │  │  - ProvisionWorkflow    │    │
│  drift_records        │           │  │  - UpdateWorkflow       │    │
└───────────────────────┘           │  │  - DecommissionWorkflow │    │
                                    │  └─────────────────────────┘    │
                                    └─────────────────────────────────┘
                                                  │
                                    ┌─────────────▼───────────────────┐
                                    │     Temporal UI 2.26            │
                                    │     localhost:8080               │
                                    └─────────────────────────────────┘
```

---

## Running Locally

### Prerequisites

- Docker & Docker Compose
- Go 1.22+
- Node.js 18+ (recommend 24 via nvm)
- Make

### Quick Start

```bash
# 1. Start infrastructure (PostgreSQL + Temporal + Temporal UI)
make up

# 2. Run database migrations
make migrate

# 3. Start the API server (terminal 1)
make run

# 4. Start the Temporal worker (terminal 2)
make worker

# 5. Start the frontend (terminal 3)
make frontend
```

### All Make Commands

| Command | Description |
|---------|-------------|
| `make up` | Start Docker services (Postgres, Temporal, Temporal UI) |
| `make down` | Stop Docker services |
| `make down-clean` | Stop services and remove volumes |
| `make logs` | Tail all service logs |
| `make build` | Build Go binaries (server + worker) |
| `make run` | Run the API server locally |
| `make worker` | Run the Temporal worker locally |
| `make frontend` | Run Next.js dev server |
| `make tidy` | Run go mod tidy |
| `make migrate` | Apply SQL migrations to Postgres |
| `make psql` | Open psql shell |
| `make clean` | Remove build artifacts |

### Access Points

| Service | URL |
|---------|-----|
| Frontend | http://localhost:3000 |
| API Server | http://localhost:8081 |
| Temporal UI | http://localhost:8080 |
| PostgreSQL | localhost:5435 |

---

## API Reference

All endpoints are prefixed with `/api/v1`. Mutations require `X-Actor` header. Team-scoped endpoints require `X-Team-ID` header.

### Teams

| Method | Path | Description |
|--------|------|-------------|
| GET | `/teams` | List all teams |
| POST | `/teams` | Create a team |
| GET | `/teams/{id}` | Get team by ID |
| DELETE | `/teams/{id}` | Delete a team |

### Environments

| Method | Path | Description |
|--------|------|-------------|
| GET | `/environments` | List environments (requires X-Team-ID) |
| POST | `/environments` | Create environment + auto-trigger Provision workflow |
| GET | `/environments/{id}` | Get environment detail |
| PUT | `/environments/{id}` | Update environment + auto-trigger Update workflow |
| DELETE | `/environments/{id}` | Delete environment |
| POST | `/environments/{id}/decommission` | Trigger Decommission workflow |

### Workflows

| Method | Path | Description |
|--------|------|-------------|
| GET | `/workflows` | List workflows (requires X-Team-ID, optional ?status=) |
| POST | `/workflows` | Create and start a workflow |
| GET | `/workflows/{id}` | Get workflow detail with steps |
| POST | `/workflows/{id}/retry` | Retry a failed/cancelled workflow |
| POST | `/workflows/{id}/cancel` | Cancel a running workflow |
| POST | `/workflows/{id}/signal` | Send approval signal (approve/reject) |

### Approvals

| Method | Path | Description |
|--------|------|-------------|
| GET | `/approvals` | List pending approvals (optional ?assigned_to=) |
| GET | `/approvals/history` | List all approvals (decision history) |
| GET | `/approvals/{id}` | Get approval detail |
| POST | `/approvals/{id}/approve` | Approve (body: {reason}) |
| POST | `/approvals/{id}/reject` | Reject (body: {reason}) |

### Drift

| Method | Path | Description |
|--------|------|-------------|
| GET | `/drift` | List drift records (optional ?environment_id=, ?unresolved=true) |
| POST | `/drift` | Report a drift record |
| GET | `/drift/{id}` | Get drift record detail |
| POST | `/drift/{id}/resolve` | Resolve drift (body: {resolution}) |

### Dashboard & Audit

| Method | Path | Description |
|--------|------|-------------|
| GET | `/dashboard/stats` | Get aggregated stats (optional ?team_id=) |
| GET | `/audit` | List audit logs (optional ?team_id=, ?limit=, ?offset=) |

---

## Workflow Steps

### Provision Workflow (10 steps)

Triggered automatically when an environment is created.

| # | Step | Type | Description |
|---|------|------|-------------|
| 1 | validate_environment | validate | Check environment config is valid |
| 2 | terraform_plan | plan | Generate infrastructure plan |
| 3 | approval_gate | approve | Pause for approval (production only) |
| 4 | create_network | apply | Create VPC and subnets |
| 5 | provision_compute | apply | Launch compute instances |
| 6 | configure_dns | apply | Set up DNS records |
| 7 | deploy_base_services | apply | Deploy monitoring, logging, ingress |
| 8 | health_checks | validate | Verify all services are healthy |
| 9 | register_catalog | notify | Register in service catalog |
| 10 | finalize | validate | Mark environment active |

### Update Workflow (6 steps)

Triggered when environment config is edited (instance size/count).

| # | Step | Type | Description |
|---|------|------|-------------|
| 1 | validate_config | validate | Validate new configuration |
| 2 | generate_plan | plan | Generate change plan |
| 3 | apply_compute | apply | Apply instance changes |
| 4 | update_dns_lb | apply | Update DNS and load balancer |
| 5 | health_checks | validate | Verify health after changes |
| 6 | finalize | validate | Mark update complete |

### Decommission Workflow (6 steps)

Triggered from the environment detail page "Decommission" button.

| # | Step | Type | Description |
|---|------|------|-------------|
| 1 | drain_traffic | apply | Drain active connections |
| 2 | deregister | apply | Remove from service catalog |
| 3 | terminate_compute | apply | Terminate instances |
| 4 | remove_dns | apply | Delete DNS records |
| 5 | delete_network | apply | Remove VPC and subnets |
| 6 | cleanup_state | apply | Clean up state, mark decommissioned |

### Failure Simulation

A 10% failure rate is applied to resource creation (network, compute, base services), update operations (compute changes, DNS/LB), and health checks. Temporal automatically retries up to 3 times with exponential backoff. If all retries fail, the workflow is marked as failed and can be retried from the UI.

### Drift Simulation

A background goroutine runs every 30 seconds and has a 10% chance per active environment of simulating drift on `instance_count`. Drift is auto-resolved when the environment is updated or decommissioned.

---

## Folder Structure

```
infraforge/
├── README.md
├── Makefile
├── docker-compose.yml
├── .env
├── .gitignore
├── temporal-config/
│   └── development-sql.yaml
├── db/
│   └── migrations/
│       └── 001_initial_schema.sql
├── backend/
│   ├── go.mod
│   ├── go.sum
│   ├── cmd/
│   │   ├── server/
│   │   │   └── main.go              # API server entry point
│   │   └── worker/
│   │       └── main.go              # Temporal worker entry point
│   └── internal/
│       ├── api/
│       │   ├── middleware.go         # CORS, logging, chain
│       │   ├── helpers.go           # JSON response, UUID parse
│       │   ├── teams.go             # Team handlers
│       │   ├── environments.go      # Environment CRUD + workflow triggers
│       │   ├── workflows.go         # Workflow list/detail/retry/cancel/signal
│       │   ├── approvals.go         # Approval list/approve/reject
│       │   ├── drift.go             # Drift list/report/resolve
│       │   ├── audit.go             # Audit log listing
│       │   └── dashboard.go         # Dashboard stats
│       ├── config/
│       │   └── config.go            # Environment-based configuration
│       ├── db/
│       │   └── db.go                # PostgreSQL connection pool
│       ├── drift/
│       │   └── simulator.go         # Background drift simulation
│       ├── models/
│       │   └── models.go            # Domain models (7 entities)
│       ├── repository/
│       │   ├── team.go
│       │   ├── environment.go
│       │   ├── workflow.go
│       │   ├── approval.go
│       │   ├── drift.go
│       │   ├── audit.go
│       │   └── dashboard.go
│       └── temporal/
│           ├── activities.go         # All workflow step activities
│           ├── workflows.go          # Provision, Update, Decommission definitions
│           └── client.go             # Temporal client + workflow starters
└── frontend/
    ├── package.json
    ├── next.config.js
    ├── tailwind.config.ts
    ├── tsconfig.json
    ├── postcss.config.js
    └── src/
        ├── lib/
        │   └── api.ts               # API client functions
        ├── components/
        │   ├── Sidebar.tsx           # Fixed left navigation
        │   ├── StatCard.tsx          # Dashboard stat card
        │   └── StatusBadge.tsx       # Color-coded status pill
        └── app/
            ├── layout.tsx            # Root layout with sidebar
            ├── globals.css           # Tailwind + dark theme
            ├── page.tsx              # Dashboard
            ├── environments/
            │   ├── page.tsx          # Environments list
            │   ├── new/page.tsx      # New environment form
            │   └── [id]/
            │       ├── page.tsx      # Environment detail + live workflow
            │       └── edit/page.tsx  # Edit instance size/count
            ├── workflows/
            │   ├── page.tsx          # Workflows list with retry
            │   └── [id]/page.tsx     # Workflow detail with steps
            ├── approvals/
            │   └── page.tsx          # Pending approvals + history
            └── drift/
                └── page.tsx          # Drift records + resolve
```

---

## Tech Stack

| Layer | Technology |
|-------|-----------|
| API Server | Go 1.22, net/http (stdlib router) |
| Workflow Engine | Temporal (Go SDK) |
| Database | PostgreSQL 16 |
| Frontend | Next.js 14, React 18, TailwindCSS 3 |
| Orchestration | Docker Compose |
| Infrastructure | Simulated (no real cloud provider calls) |

---

## Architecture Decisions & Trade-offs

| Decision | Rationale | Trade-off |
|----------|-----------|-----------|
| **Go stdlib router** (no framework) | Zero dependencies, Go 1.22 pattern matching is sufficient | Less middleware ecosystem than Chi/Echo, but simpler to understand |
| **Temporal for workflows** | Durable execution, built-in retries, signal-based approval gates, visibility into step state | Adds infrastructure complexity (Temporal server + worker process) |
| **Polling (not WebSockets)** | Simpler to implement, no connection state to manage, works through proxies | Slightly higher latency (2s) and more HTTP requests than push-based |
| **Single Postgres** for app + Temporal | Simpler local setup, fewer containers | In production you'd want separate databases |
| **Simulated infrastructure** | Demonstrates the orchestration pattern without cloud credentials | No real resources are created — activities use `time.Sleep` |
| **Team-scoped via headers** | Simple multi-tenancy without auth complexity | Real app would use JWT/OAuth with team claims |
| **10% failure simulator** | Demonstrates retry behavior and failure handling realistically | Can be confusing if you don't know it's intentional |
| **Drift as background goroutine** | Runs in the API server process, no extra deployment | In production this would be a separate scheduled Temporal workflow |
| **Pre-trigger workflows on create/update** | User sees immediate feedback without manual "Deploy" click | Couples environment CRUD tightly to workflow orchestration |

