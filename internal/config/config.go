package config

// Config holds all runtime configuration parsed by Kong.
type Config struct {
	Port        int    `default:"8080"               env:"PORT"         help:"HTTP listen port"`
	DatabaseURL string `default:"./data/pipeline.db" env:"DATABASE_URL" help:"SQLite path or Postgres DSN"`
	OutputDir   string `default:"./output"           env:"OUTPUT_DIR"   help:"Directory for generated resumes and cover letters"`
	DataDir     string `default:"./data"             env:"DATA_DIR"     help:"Directory for the database file"`
}
