package cmd

import (
	"database/sql"
	"fmt"
	"go/types"
	"log"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stellar/go/services/horizon/internal/db2/schema"
	"github.com/stellar/go/services/horizon/internal/expingest"
	support "github.com/stellar/go/support/config"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	hlog "github.com/stellar/go/support/log"
)

var dbCmd = &cobra.Command{
	Use:   "db [command]",
	Short: "commands to manage horizon's postgres db",
}

var dbInitCmd = &cobra.Command{
	Use:   "init",
	Short: "install schema",
	Long:  "init initializes the postgres database used by horizon.",
	Run: func(cmd *cobra.Command, args []string) {
		dbURLConfigOption.Require()
		dbURLConfigOption.SetValue()

		db, err := sql.Open("postgres", viper.GetString("db-url"))
		if err != nil {
			log.Fatal(err)
		}

		numMigrationsRun, err := schema.Migrate(db, schema.MigrateUp, 0)
		if err != nil {
			log.Fatal(err)
		}

		if numMigrationsRun == 0 {
			log.Println("No migrations applied.")
		} else {
			log.Printf("Successfully applied %d migrations.\n", numMigrationsRun)
		}
	},
}

var dbMigrateCmd = &cobra.Command{
	Use:   "migrate [up|down|redo] [COUNT]",
	Short: "migrate schema",
	Long:  "performs a schema migration command",
	Run: func(cmd *cobra.Command, args []string) {
		// Allow invokations with 1 or 2 args.  All other args counts are erroneous.
		if len(args) < 1 || len(args) > 2 {
			cmd.Usage()
			os.Exit(1)
		}

		dir := schema.MigrateDir(args[0])
		count := 0

		// If a second arg is present, parse it to an int and use it as the count
		// argument to the migration call.
		if len(args) == 2 {
			var err error
			count, err = strconv.Atoi(args[1])
			if err != nil {
				log.Println(err)
				cmd.Usage()
				os.Exit(1)
			}
		}

		dbURLConfigOption.Require()
		dbURLConfigOption.SetValue()

		db, err := sql.Open("postgres", viper.GetString("db-url"))
		if err != nil {
			log.Fatal(err)
		}
		pingDB(db)

		numMigrationsRun, err := schema.Migrate(db, dir, count)
		if err != nil {
			log.Fatal(err)
		}

		if numMigrationsRun == 0 {
			log.Println("No migrations applied.")
		} else {
			log.Printf("Successfully applied %d migrations.\n", numMigrationsRun)
		}
	},
}

var dbReapCmd = &cobra.Command{
	Use:   "reap",
	Short: "reaps (i.e. removes) any reapable history data",
	Long:  "reap removes any historical data that is earlier than the configured retention cutoff",
	Run: func(cmd *cobra.Command, args []string) {
		app := initApp()
		app.UpdateLedgerState()
		err := app.DeleteUnretainedHistory()
		if err != nil {
			log.Fatal(err)
		}
	},
}

var dbReingestCmd = &cobra.Command{
	Use:   "reingest",
	Short: "reingest commands",
	Long:  "reingest ingests historical data for every ledger or ledgers specified by subcommand",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Use one of the subcomands...")
		cmd.Usage()
		os.Exit(1)
	},
}

var reingestForce bool
var reingestRangeCmdOpts = []*support.ConfigOption{
	&support.ConfigOption{
		Name:        "force",
		ConfigKey:   &reingestForce,
		OptType:     types.Bool,
		Required:    false,
		FlagDefault: false,
		Usage: "[optional] if this flag is set, horizon will be blocked " +
			"from ingesting until the reingestion command completes",
	},
}

var dbReingestRangeCmd = &cobra.Command{
	Use:   "range [Start sequence number] [End sequence number]",
	Short: "reingests ledgers within a range",
	Long:  "reingests ledgers between X and Y sequence number (closed intervals)",
	Run: func(cmd *cobra.Command, args []string) {
		for _, co := range reingestRangeCmdOpts {
			co.Require()
			co.SetValue()
		}

		if len(args) != 2 {
			cmd.Usage()
			os.Exit(1)
		}

		argsInt32 := make([]uint32, 2)
		for i, arg := range args {
			seq, err := strconv.Atoi(arg)
			if err != nil {
				cmd.Usage()
				log.Fatalf(`Invalid sequence number "%s"`, arg)
			}
			argsInt32[i] = uint32(seq)
		}

		initRootConfig()

		coreSession, err := db.Open("postgres", config.StellarCoreDatabaseURL)
		if err != nil {
			log.Fatalf("cannot open Core DB: %v", err)
		}

		horizonSession, err := db.Open("postgres", config.DatabaseURL)
		if err != nil {
			log.Fatalf("cannot open Horizon DB: %v", err)
		}

		ingestConfig := expingest.Config{
			CoreSession:              coreSession,
			NetworkPassphrase:        config.NetworkPassphrase,
			HistorySession:           horizonSession,
			HistoryArchiveURL:        config.HistoryArchiveURLs[0],
			IngestFailedTransactions: config.IngestFailedTransactions,
		}

		system, err := expingest.NewSystem(ingestConfig)
		if err != nil {
			log.Fatal(err)
		}

		err = system.ReingestRange(
			argsInt32[0],
			argsInt32[1],
			reingestForce,
		)
		if err == nil {
			hlog.Info("Range run successfully!")
			return
		}

		if errors.Cause(err) == expingest.ErrReingestRangeConflict {
			message := `
			The range you have provided overlaps with Horizon's most recently ingested ledger.
			It is not possible to run the reingest command on this range in parallel with
			Horizon's ingestion system.
			Either reduce the range so that it doesn't overlap with Horizon's ingestion system,
			or, use the force flag to ensure that Horizon's ingestion system is blocked until
			the reingest command completes.
			`
			log.Fatal(message)
		}

		log.Fatal(err)
	},
}

func init() {
	for _, co := range reingestRangeCmdOpts {
		err := co.Init(dbReingestRangeCmd)
		if err != nil {
			log.Fatal(err.Error())
		}
	}

	viper.BindPFlags(dbReingestRangeCmd.PersistentFlags())

	rootCmd.AddCommand(dbCmd)
	dbCmd.AddCommand(
		dbInitCmd,
		dbMigrateCmd,
		dbReapCmd,
		dbReingestCmd,
	)
	dbReingestCmd.AddCommand(dbReingestRangeCmd)
}
