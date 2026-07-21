package config

// Config holds all runtime configuration parsed by Kong.
type Config struct {
	Port        int    `env:"PORT"         default:"8080"        help:"HTTP listen port"`
	DatabaseURL string `env:"DATABASE_URL"  default:"./data/pipeline.db" help:"SQLite path or Postgres DSN"`
	OutputDir   string `env:"OUTPUT_DIR"    default:"./output"    help:"Directory for generated resumes and cover letters"`
	DataDir     string `env:"DATA_DIR"      default:"./data"      help:"Directory for the database file"`
}
