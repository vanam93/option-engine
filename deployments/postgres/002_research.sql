-- Stage 4 Phase 6: Research Repository schema

INSERT INTO schema_migrations (version) VALUES (2) ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS research_experiments (
    experiment_id   TEXT PRIMARY KEY,
    strategy        TEXT NOT NULL,
    symbol          TEXT NOT NULL,
    timeframe       TEXT NOT NULL,
    parameters      JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS optimization_results (
    id              BIGSERIAL PRIMARY KEY,
    experiment_id   TEXT NOT NULL REFERENCES research_experiments(experiment_id) ON DELETE CASCADE,
    score           DOUBLE PRECISION NOT NULL,
    win_rate        DOUBLE PRECISION NOT NULL,
    expectancy      DOUBLE PRECISION NOT NULL,
    profit_factor   DOUBLE PRECISION NOT NULL,
    drawdown        DOUBLE PRECISION NOT NULL,
    metrics         JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_optimization_results_experiment
    ON optimization_results(experiment_id);

CREATE TABLE IF NOT EXISTS walkforward_results (
    id                  BIGSERIAL PRIMARY KEY,
    walkforward_id      TEXT NOT NULL,
    experiment_id       TEXT NOT NULL REFERENCES research_experiments(experiment_id) ON DELETE CASCADE,
    run_id              TEXT NOT NULL,
    train_score         DOUBLE PRECISION NOT NULL,
    validation_score    DOUBLE PRECISION NOT NULL,
    parameter_set       JSONB NOT NULL DEFAULT '{}',
    performance_metrics JSONB NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (walkforward_id, experiment_id, run_id)
);

CREATE INDEX IF NOT EXISTS idx_walkforward_results_experiment
    ON walkforward_results(experiment_id);

CREATE TABLE IF NOT EXISTS montecarlo_results (
    simulation_id       TEXT PRIMARY KEY,
    walkforward_id      TEXT NOT NULL,
    experiment_id       TEXT NOT NULL REFERENCES research_experiments(experiment_id) ON DELETE CASCADE,
    simulations         INT NOT NULL,
    confidence_interval JSONB NOT NULL DEFAULT '{}',
    probability_of_profit DOUBLE PRECISION NOT NULL,
    probability_of_loss   DOUBLE PRECISION NOT NULL,
    risk_of_ruin        DOUBLE PRECISION NOT NULL,
    distribution        JSONB NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_montecarlo_results_experiment
    ON montecarlo_results(experiment_id);

CREATE TABLE IF NOT EXISTS research_reports (
    research_id     TEXT PRIMARY KEY,
    experiment_id   TEXT NOT NULL REFERENCES research_experiments(experiment_id) ON DELETE CASCADE,
    version         INT NOT NULL DEFAULT 1,
    json_path       TEXT,
    csv_path        TEXT,
    generated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_research_experiments_strategy
    ON research_experiments(strategy);

CREATE INDEX IF NOT EXISTS idx_research_experiments_symbol
    ON research_experiments(symbol);

CREATE INDEX IF NOT EXISTS idx_research_experiments_timeframe
    ON research_experiments(timeframe);

CREATE INDEX IF NOT EXISTS idx_research_reports_experiment
    ON research_reports(experiment_id);
