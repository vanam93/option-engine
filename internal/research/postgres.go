package research

import (
	"context"
	"embed"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var schemaEmbed embed.FS

func loadSchemaSQL() (string, error) {
	data, err := schemaEmbed.ReadFile("schema.sql")
	if err != nil {
		return "", fmt.Errorf("load schema: %w", err)
	}
	return string(data), nil
}

// EnsureSchema applies research DDL idempotently.
func (r *PostgresRepository) EnsureSchema(ctx context.Context) error {
	schemaSQL, err := loadSchemaSQL()
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, schemaSQL)
	if err != nil {
		return fmt.Errorf("research schema: %w", err)
	}
	return nil
}

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a PostgreSQL-backed research repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) EnsureExperiment(ctx context.Context, exp ResearchExperiment) error {
	if exp.Parameters == nil {
		exp.Parameters = []byte("{}")
	}
	createdAt := exp.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO research_experiments (experiment_id, strategy, symbol, timeframe, parameters, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (experiment_id) DO NOTHING
	`, exp.ExperimentID, exp.Strategy, exp.Symbol, exp.Timeframe, exp.Parameters, createdAt)
	return err
}

func (r *PostgresRepository) UpsertExperiment(ctx context.Context, exp ResearchExperiment) error {
	if exp.Parameters == nil {
		exp.Parameters = []byte("{}")
	}
	createdAt := exp.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO research_experiments (experiment_id, strategy, symbol, timeframe, parameters, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (experiment_id) DO UPDATE SET
			strategy = EXCLUDED.strategy,
			symbol = EXCLUDED.symbol,
			timeframe = EXCLUDED.timeframe,
			parameters = EXCLUDED.parameters
	`, exp.ExperimentID, exp.Strategy, exp.Symbol, exp.Timeframe, exp.Parameters, createdAt)
	return err
}

func (r *PostgresRepository) InsertOptimizationResult(ctx context.Context, result OptimizationResult) error {
	if result.Metrics == nil {
		result.Metrics = []byte("{}")
	}
	createdAt := result.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO optimization_results
			(experiment_id, score, win_rate, expectancy, profit_factor, drawdown, metrics, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, result.ExperimentID, result.Score, result.WinRate, result.Expectancy,
		result.ProfitFactor, result.Drawdown, result.Metrics, createdAt)
	return err
}

func (r *PostgresRepository) InsertWalkForwardResult(ctx context.Context, result WalkForwardResult) error {
	if result.ParameterSet == nil {
		result.ParameterSet = []byte("{}")
	}
	if result.PerformanceMetrics == nil {
		result.PerformanceMetrics = []byte("{}")
	}
	createdAt := result.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO walkforward_results
			(walkforward_id, experiment_id, run_id, train_score, validation_score, parameter_set, performance_metrics, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (walkforward_id, experiment_id, run_id) DO UPDATE SET
			train_score = EXCLUDED.train_score,
			validation_score = EXCLUDED.validation_score,
			parameter_set = EXCLUDED.parameter_set,
			performance_metrics = EXCLUDED.performance_metrics
	`, result.WalkForwardID, result.ExperimentID, result.RunID,
		result.TrainScore, result.ValidationScore, result.ParameterSet, result.PerformanceMetrics, createdAt)
	return err
}

func (r *PostgresRepository) InsertMonteCarloResult(ctx context.Context, result MonteCarloResult) error {
	if result.ConfidenceInterval == nil {
		result.ConfidenceInterval = []byte("{}")
	}
	if result.Distribution == nil {
		result.Distribution = []byte("{}")
	}
	createdAt := result.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO montecarlo_results
			(simulation_id, walkforward_id, experiment_id, simulations,
			 confidence_interval, probability_of_profit, probability_of_loss, risk_of_ruin, distribution, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (simulation_id) DO UPDATE SET
			confidence_interval = EXCLUDED.confidence_interval,
			probability_of_profit = EXCLUDED.probability_of_profit,
			probability_of_loss = EXCLUDED.probability_of_loss,
			risk_of_ruin = EXCLUDED.risk_of_ruin,
			distribution = EXCLUDED.distribution
	`, result.SimulationID, result.WalkForwardID, result.ExperimentID, result.Simulations,
		result.ConfidenceInterval, result.ProbabilityOfProfit, result.ProbabilityOfLoss,
		result.RiskOfRuin, result.Distribution, createdAt)
	return err
}

func (r *PostgresRepository) GetExperiment(ctx context.Context, experimentID string) (ResearchExperiment, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT experiment_id, strategy, symbol, timeframe, parameters, created_at
		FROM research_experiments
		WHERE experiment_id = $1
	`, experimentID)

	var exp ResearchExperiment
	if err := row.Scan(&exp.ExperimentID, &exp.Strategy, &exp.Symbol, &exp.Timeframe, &exp.Parameters, &exp.CreatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return ResearchExperiment{}, ErrNotFound
		}
		return ResearchExperiment{}, err
	}
	return exp, nil
}

func (r *PostgresRepository) ListExperiments(ctx context.Context, filter QueryFilter) ([]ResearchExperiment, error) {
	query := `
		SELECT experiment_id, strategy, symbol, timeframe, parameters, created_at
		FROM research_experiments
		WHERE 1=1`
	args := make([]any, 0, 4)
	argPos := 1

	if filter.ExperimentID != "" {
		query += fmt.Sprintf(" AND experiment_id = $%d", argPos)
		args = append(args, filter.ExperimentID)
		argPos++
	}
	if filter.Strategy != "" {
		query += fmt.Sprintf(" AND strategy = $%d", argPos)
		args = append(args, filter.Strategy)
		argPos++
	}
	if filter.Symbol != "" {
		query += fmt.Sprintf(" AND symbol = $%d", argPos)
		args = append(args, filter.Symbol)
		argPos++
	}
	if filter.Timeframe != "" {
		query += fmt.Sprintf(" AND timeframe = $%d", argPos)
		args = append(args, filter.Timeframe)
		argPos++
	}
	query += " ORDER BY created_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]ResearchExperiment, 0)
	for rows.Next() {
		var exp ResearchExperiment
		if err := rows.Scan(&exp.ExperimentID, &exp.Strategy, &exp.Symbol, &exp.Timeframe, &exp.Parameters, &exp.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, exp)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) GetResearchBundle(ctx context.Context, experimentID string) (ResearchBundle, error) {
	exp, err := r.GetExperiment(ctx, experimentID)
	if err != nil {
		return ResearchBundle{}, err
	}

	optimization, err := r.listOptimization(ctx, experimentID)
	if err != nil {
		return ResearchBundle{}, err
	}
	walkforward, err := r.listWalkForward(ctx, experimentID)
	if err != nil {
		return ResearchBundle{}, err
	}
	montecarlo, err := r.listMonteCarlo(ctx, experimentID)
	if err != nil {
		return ResearchBundle{}, err
	}
	reports, err := r.listReports(ctx, experimentID)
	if err != nil {
		return ResearchBundle{}, err
	}

	return ResearchBundle{
		Experiment:   exp,
		Optimization: optimization,
		WalkForward:  walkforward,
		MonteCarlo:   montecarlo,
		Reports:      reports,
	}, nil
}

func (r *PostgresRepository) listOptimization(ctx context.Context, experimentID string) ([]OptimizationResult, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, experiment_id, score, win_rate, expectancy, profit_factor, drawdown, metrics, created_at
		FROM optimization_results
		WHERE experiment_id = $1
		ORDER BY created_at ASC
	`, experimentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]OptimizationResult, 0)
	for rows.Next() {
		var row OptimizationResult
		if err := rows.Scan(&row.ID, &row.ExperimentID, &row.Score, &row.WinRate, &row.Expectancy,
			&row.ProfitFactor, &row.Drawdown, &row.Metrics, &row.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) listWalkForward(ctx context.Context, experimentID string) ([]WalkForwardResult, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, walkforward_id, experiment_id, run_id, train_score, validation_score,
		       parameter_set, performance_metrics, created_at
		FROM walkforward_results
		WHERE experiment_id = $1
		ORDER BY created_at ASC
	`, experimentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]WalkForwardResult, 0)
	for rows.Next() {
		var row WalkForwardResult
		if err := rows.Scan(&row.ID, &row.WalkForwardID, &row.ExperimentID, &row.RunID,
			&row.TrainScore, &row.ValidationScore, &row.ParameterSet, &row.PerformanceMetrics, &row.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) listMonteCarlo(ctx context.Context, experimentID string) ([]MonteCarloResult, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT simulation_id, walkforward_id, experiment_id, simulations,
		       confidence_interval, probability_of_profit, probability_of_loss, risk_of_ruin, distribution, created_at
		FROM montecarlo_results
		WHERE experiment_id = $1
		ORDER BY created_at ASC
	`, experimentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]MonteCarloResult, 0)
	for rows.Next() {
		var row MonteCarloResult
		if err := rows.Scan(&row.SimulationID, &row.WalkForwardID, &row.ExperimentID, &row.Simulations,
			&row.ConfidenceInterval, &row.ProbabilityOfProfit, &row.ProbabilityOfLoss,
			&row.RiskOfRuin, &row.Distribution, &row.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) listReports(ctx context.Context, experimentID string) ([]ResearchReport, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT research_id, experiment_id, version, json_path, csv_path, generated_at
		FROM research_reports
		WHERE experiment_id = $1
		ORDER BY generated_at ASC
	`, experimentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]ResearchReport, 0)
	for rows.Next() {
		var row ResearchReport
		if err := rows.Scan(&row.ResearchID, &row.ExperimentID, &row.Version,
			&row.JSONPath, &row.CSVPath, &row.GeneratedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) InsertResearchReport(ctx context.Context, report ResearchReport) error {
	generatedAt := report.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO research_reports (research_id, experiment_id, version, json_path, csv_path, generated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, report.ResearchID, report.ExperimentID, report.Version, report.JSONPath, report.CSVPath, generatedAt)
	return err
}

func (r *PostgresRepository) LatestReportVersion(ctx context.Context, experimentID string) (int, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(version), 0)
		FROM research_reports
		WHERE experiment_id = $1
	`, experimentID)
	var version int
	if err := row.Scan(&version); err != nil {
		return 0, err
	}
	return version, nil
}

func (r *PostgresRepository) CountEntries(ctx context.Context) (int64, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM research_experiments) +
			(SELECT COUNT(*) FROM optimization_results) +
			(SELECT COUNT(*) FROM walkforward_results) +
			(SELECT COUNT(*) FROM montecarlo_results) +
			(SELECT COUNT(*) FROM research_reports)
	`)
	var count int64
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}
