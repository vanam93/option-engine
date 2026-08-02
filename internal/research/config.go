package research

import "fmt"

// Config controls the research repository and reporting engine.
type Config struct {
	Enabled          bool
	ExportDirectory  string
	Formats          []string
	SubscriberBuffer int
}

func (c Config) withDefaults() Config {
	out := c
	if out.ExportDirectory == "" {
		out.ExportDirectory = "./reports"
	}
	if len(out.Formats) == 0 {
		out.Formats = []string{"json", "csv"}
	}
	if out.SubscriberBuffer <= 0 {
		out.SubscriberBuffer = 256
	}
	return out
}

// Validate returns an error when configuration is unusable.
func (c Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.ExportDirectory == "" {
		return fmt.Errorf("research: export_directory is required")
	}
	if c.SubscriberBuffer < 1 {
		return fmt.Errorf("research: subscriber_buffer must be >= 1")
	}
	for _, format := range c.Formats {
		if format != "json" && format != "csv" {
			return fmt.Errorf("research: unsupported format %q", format)
		}
	}
	return nil
}
