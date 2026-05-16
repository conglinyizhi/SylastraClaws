package cron

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/conglinyizhi/SylastraClaws/internal"
	pkgcron "github.com/conglinyizhi/SylastraClaws/pkg/cron"
)

func NewCronCommand() *cobra.Command {
	var storePath string

	cmd := &cobra.Command{
		Use:     "cron",
		Aliases: []string{"c"},
		Short:   "Manage scheduled tasks",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := internal.LoadConfig()
			if err != nil {
				return fmt.Errorf("error loading config: %w", err)
			}
			storePath = filepath.Join(cfg.WorkspacePath(), "cron", "jobs.json")
			return nil
		},
	}

	cmd.AddCommand(
		newListCommand(func() string { return storePath }),
		newAddCommand(func() string { return storePath }),
		newRemoveCommand(func() string { return storePath }),
		newEnableCommand(func() string { return storePath }),
		newDisableCommand(func() string { return storePath }),
	)

	return cmd
}

func newListCommand(storePath func() string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all scheduled jobs",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			cronListCmd(storePath())
			return nil
		},
	}
}

func newRemoveCommand(storePath func() string) *cobra.Command {
	return &cobra.Command{
		Use:     "remove",
		Short:   "Remove a job by ID",
		Args:    cobra.ExactArgs(1),
		Example: "sylastraclaws cron remove 1",
		RunE: func(_ *cobra.Command, args []string) error {
			cronRemoveCmd(storePath(), args[0])
			return nil
		},
	}
}

func newEnableCommand(storePath func() string) *cobra.Command {
	return &cobra.Command{
		Use:     "enable",
		Short:   "Enable a job",
		Args:    cobra.ExactArgs(1),
		Example: "sylastraclaws cron enable 1",
		RunE: func(_ *cobra.Command, args []string) error {
			cronSetJobEnabled(storePath(), args[0], true)
			return nil
		},
	}
}

func newDisableCommand(storePath func() string) *cobra.Command {
	return &cobra.Command{
		Use:     "disable",
		Short:   "Disable a job",
		Args:    cobra.ExactArgs(1),
		Example: "sylastraclaws cron disable 1",
		RunE: func(_ *cobra.Command, args []string) error {
			cronSetJobEnabled(storePath(), args[0], false)
			return nil
		},
	}
}

func cronListCmd(storePath string) {
	cs := pkgcron.NewCronService(storePath, nil)
	jobs := cs.ListJobs(true)

	if len(jobs) == 0 {
		fmt.Println("No scheduled jobs.")
		return
	}

	fmt.Println("\nScheduled Jobs:")
	fmt.Println("----------------")
	for _, job := range jobs {
		var schedule string
		if job.Schedule.Kind == "every" && job.Schedule.EveryMS != nil {
			schedule = fmt.Sprintf("every %ds", *job.Schedule.EveryMS/1000)
		} else if job.Schedule.Kind == "cron" {
			schedule = job.Schedule.Expr
		} else {
			schedule = "one-time"
		}

		nextRun := "scheduled"
		if job.State.NextRunAtMS != nil {
			nextTime := time.UnixMilli(*job.State.NextRunAtMS)
			nextRun = nextTime.Format("2006-01-02 15:04")
		}

		status := "enabled"
		if !job.Enabled {
			status = "disabled"
		}

		fmt.Printf("  %s (%s)\n", job.Name, job.ID)
		fmt.Printf("    Schedule: %s\n", schedule)
		fmt.Printf("    Status: %s\n", status)
		fmt.Printf("    Next run: %s\n", nextRun)
	}
}

func cronRemoveCmd(storePath, jobID string) {
	cs := pkgcron.NewCronService(storePath, nil)
	if cs.RemoveJob(jobID) {
		fmt.Printf("✓ Removed job %s\n", jobID)
	} else {
		fmt.Printf("✗ Job %s not found\n", jobID)
	}
}

func cronSetJobEnabled(storePath, jobID string, enabled bool) {
	cs := pkgcron.NewCronService(storePath, nil)
	job := cs.EnableJob(jobID, enabled)
	if job != nil {
		fmt.Printf("✓ Job '%s' enabled\n", job.Name)
	} else {
		fmt.Printf("✗ Job %s not found\n", jobID)
	}
}
